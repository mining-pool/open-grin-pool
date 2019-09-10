package main

// http rpc server
import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type stratumServer struct {
	db   *badger.DB
	ln   net.Listener
	conf *config
}

type stratumRequest struct {
	ID      string                 `json:"id"`
	JsonRpc string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
}

type stratumResponse struct {
	ID      string                 `json:"id"`
	JsonRpc string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Result  interface{}            `json:"result"`
	Error   map[string]interface{} `json:"error"`
}

func (ss *stratumServer) handle(conn net.Conn) {
	startTime := time.Now()
	defer conn.Close()
	var login string
	nc := initNodeStratumClient(ss.conf)

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	go func() {
		statusReq := &stratumRequest{
			ID:      "0",
			JsonRpc: "2.0",
			Method:  "status",
			Params:  nil,
		}

		ch := time.Tick(10 * time.Second)
		enc := json.NewEncoder(nc.c)

		for {
			select {
			case <-ch:
				err := enc.Encode(statusReq)
				if err != nil {
					log.Println(err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go nc.wait(func(sr json.RawMessage) {
		enc := json.NewEncoder(conn)
		err := enc.Encode(sr)
		if err != nil {
			log.Println(err)
		}

		// internal record
		var res stratumResponse
		_ = json.Unmarshal(sr, &res) // suppress the err

		if res.Method == "status" && login != "" {
			result, _ := res.Result.(map[string]interface{})
			uptime := uint64(time.Since(startTime).Seconds())
			result["uptime"] = uptime

			var shareNum uint64
			err := ss.db.View(func(txn *badger.Txn) error {
				item, err := txn.Get([]byte("shares+" + login))
				if err != nil {
					return err
				}

				raw, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				shareNum = new(big.Int).SetBytes(raw).Uint64()

				return nil
			})

			hs := float64(shareNum) / float64(uptime)
			result["hashrate"] = strconv.FormatFloat(hs, 'f', 2, 64) + "Sol/s"
			status, _ := json.Marshal(result)
			if err != nil {
				log.Println(err)
			}

			err = ss.db.Update(func(txn *badger.Txn) error {
				e := badger.NewEntry([]byte("status"+login), status).WithTTL(time.Hour)
				err := txn.SetEntry(e)
				return err
			})
			if err != nil {
				log.Println(err)
			}
		}

		if res.Method == "submit" {
			detail, ok := res.Result.(string)
			log.Println(login, "has submit a", detail, "share")
			if ok {
				if strings.Contains(detail, "block - ") {
					blockHash := strings.Trim(detail, "block - ")
					var blockBatch []string
					err := ss.db.Update(func(txn *badger.Txn) error {
						item, err := txn.Get([]byte("blocks"))
						if err != nil {
							return err
						}
						raw, err := item.ValueCopy(nil)
						err = json.Unmarshal(raw, &blockBatch)
						if err != nil {
							return err
						}
						return nil
					})
					if err != nil {
						log.Println(err)
					}

					blockBatch = append(blockBatch, blockHash)

					err = ss.db.Update(func(txn *badger.Txn) error {
						raw, _ := json.Marshal(blockBatch)
						if err != nil {
							log.Println(err)
						}
						err = txn.Set([]byte("blocks"), raw)
						return err
					})
					if err != nil {
						log.Println(err)
					}
				}
			}
		}
	})
	defer nc.c.Close()

	dec := json.NewDecoder(conn)
	for {
		var jsonRaw json.RawMessage
		var clientReq stratumRequest

		err := dec.Decode(&jsonRaw)
		if err != nil {
			log.Println("failed to decode 2 raw", err)
			return
		}

		err = json.Unmarshal(jsonRaw, &clientReq)
		if err != nil {
			log.Println("failed to decode 2 clientReq", err)
			return
		}

		switch clientReq.Method {
		case "login":
			var ok bool
			login, ok = clientReq.Params["login"].(string)
			if !ok {
				log.Println("login module broken")
			}

			pass, ok := clientReq.Params["pass"].(string)
			if !ok {
				log.Println("login module broken")
			}

			login = strings.TrimSpace(login)
			pass = strings.TrimSpace(pass)

			if strings.Contains(login, "+") {
				return
			}

			isRegistered := true
			err := ss.db.View(func(txn *badger.Txn) error {
				item, err := txn.Get([]byte("login+" + login))
				if err != nil {
					if err == badger.ErrKeyNotFound {
						isRegistered = false
						return nil
					}
					return err
				}

				if item.ValueSize() == 0 {
					isRegistered = false
					return nil
				}

				passInDB, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				if string(passInDB) != pass {
					return errWrongPassword
				}

				return err
			})
			if err != nil {
				login = ""
				log.Println(err)
			}
			if err == errWrongPassword {
				return
			}

			if isRegistered == false {
				err := ss.db.Update(func(txn *badger.Txn) error {
					err = txn.Set([]byte("login+"+login), []byte(pass))
					return err
				})
				if err != nil {
					login = ""
					log.Println(err)
				}
			}

			log.Println(login, "has logged in")
			go relay2Node(nc, jsonRaw)
			break

		case "submit":
			var lastShareCount uint64

			err := ss.db.View(func(txn *badger.Txn) error {
				item, err := txn.Get([]byte("shares+" + login))
				if err != nil {
					if err == badger.ErrKeyNotFound {
						lastShareCount = 0
						return nil
					}
					return err
				}

				bCount, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}

				lastShareCount = new(big.Int).SetBytes(bCount).Uint64()

				return nil
			})
			if err != nil {
				log.Println(err)
			}

			lastShareCount++

			err = ss.db.Update(func(txn *badger.Txn) error {
				e := badger.NewEntry([]byte("shares+"+login), new(big.Int).SetUint64(lastShareCount).Bytes()).WithTTL(time.Hour)
				err := txn.SetEntry(e)
				return err
			})
			if err != nil {
				log.Println(err)
			}

			go relay2Node(nc, jsonRaw)
			break

		case "getjobtemplate":
		case "job":
		case "keepalive":
		case "height":
		default:
			go relay2Node(nc, jsonRaw)
		}
	}
}

func relay2Node(nc *nodeClient, data json.RawMessage) {
	enc := json.NewEncoder(nc.c)
	err := enc.Encode(data)
	if err != nil {
		log.Println("relay2Node", err)
	}
}

func initStratumServer(db *badger.DB, conf *config) {
	ln, err := net.Listen("tcp",
		conf.StratumServer.Address+":"+strconv.Itoa(conf.StratumServer.Port),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("listening on ", conf.StratumServer.Port)

	ss := &stratumServer{
		db:   db,
		ln:   ln,
		conf: conf,
	}

	go ss.backupPerInterval()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
		}
		log.Println("new conn from", conn.RemoteAddr())
		go ss.handle(conn)
	}
}

func (ss *stratumServer) backupPerInterval() {
	d, err := time.ParseDuration(ss.conf.StratumServer.BackupInterval)
	if err != nil {
		log.Println("failed to start export system", err)
		return
	}

	log.Println("export system running")

	ch := time.Tick(d)
	for {
		select {
		case <-ch:
			newFileName := strconv.Itoa(time.Now().Year()) + "-" +
				time.Now().Month().String() + "-" + strconv.Itoa(time.Now().Day()) +
				"-" + strconv.Itoa(time.Now().Hour())
			f, err := os.Create(newFileName + ".csv")
			if err != nil {
				log.Println(err)
				continue
			}
			_ = ss.db.View(func(txn *badger.Txn) error {
				it := txn.NewIterator(badger.DefaultIteratorOptions)
				defer it.Close()
				prefix := []byte("shares+")
				for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
					item := it.Item()
					k := item.Key()
					_ = item.Value(func(v []byte) error {
						_, err = fmt.Fprintf(f, "%s %d\n", k, new(big.Int).SetBytes(v).Uint64())
						log.Println(err)
						return nil
					})
				}

				return nil
			})
			_ = f.Close()
		}
	}
}
