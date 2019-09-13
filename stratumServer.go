package main

// http rpc server
import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/google/logger"
)

type stratumServer struct {
	db   *database
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

type minerSession struct {
	login      string
	difficulty int64
	ctx        context.Context
}

func (ms *minerSession) hasLoggedIn() bool {
	return ms.login == ""
}

func (ms *minerSession) handleMethod(res *stratumResponse, db *database) {
	switch res.Method {
	case "status":
		if ms.login == "" {
			logger.Warning("recv status detail before login")
			break
		}
		result, _ := res.Result.(map[string]interface{})
		db.setMinerStatus(ms.login, result)
		ms.difficulty, _ = result["difficulty"].(int64)

		break
	case "submit":
		db.putShare(ms.login, ms.difficulty)
		if res.Error != nil {
			logger.Warning(ms.login, "'s share has err:", res.Error)
			break
		}
		detail, ok := res.Result.(string)
		logger.Info(ms.login, "has submit a", detail, "share")
		if ok {
			if strings.Contains(detail, "block - ") {
				blockHash := strings.Trim(detail, "block - ")
				db.putBlockHash(blockHash)
				logger.Warning("block", blockHash, "has been found by", ms.login)
			}
		}
		break
	}
}

func callStatusPerInterval(ctx context.Context, nc *nodeClient) {
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
				logger.Error(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (ss *stratumServer) handleConn(conn net.Conn) {
	logger.Info("new conn from ", conn.RemoteAddr())
	session := &minerSession{difficulty: 1}
	defer conn.Close()
	var login string
	nc := initNodeStratumClient(ss.conf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go callStatusPerInterval(ctx, nc)

	go nc.registerHandler(ctx, func(sr json.RawMessage) {
		enc := json.NewEncoder(conn)
		err := enc.Encode(sr)
		if err != nil {
			logger.Error(err)
		}

		// internal record
		var res stratumResponse
		_ = json.Unmarshal(sr, &res) // suppress the err

		session.handleMethod(&res, ss.db)
	})
	defer nc.close()

	dec := json.NewDecoder(conn)
	for {
		var jsonRaw json.RawMessage
		var clientReq stratumRequest

		err := dec.Decode(&jsonRaw)
		if err != nil {
			logger.Error(err)
			return
		}

		err = json.Unmarshal(jsonRaw, &clientReq)
		if err != nil {
			logger.Error(err)
			return
		}

		logger.Info(conn.RemoteAddr(), " sends a ", clientReq.Method, " request:", string(jsonRaw))

		switch clientReq.Method {
		case "login":
			var ok bool
			login, ok = clientReq.Params["login"].(string)
			if !ok {
				logger.Error("login module broken")
				return
			}

			pass, ok := clientReq.Params["pass"].(string)
			if !ok {
				logger.Error("login module broken")
				return
			}

			login = strings.TrimSpace(login)
			pass = strings.TrimSpace(pass)

			switch ss.db.verifyMiner(login, pass) {
			case wrongPassword:
				logger.Warning(login, " has failed to login")
				return
			case noPassword:
				ss.db.registerMiner(login, pass, "")
				logger.Warning(login, " has registered")
			case correctPassword:
			}

			session.login = login
			logger.Info(session.login, " has logged in")
			go relay2Node(nc, jsonRaw)
			break

		case "submit": // migrate to the resp handler
		case "getjobtemplate":
		case "job":
		case "keepalive":
		case "height":
		default:
			if !session.hasLoggedIn() {
				logger.Warning(login, " has not logged in")
			}

			go relay2Node(nc, jsonRaw)
		}
	}
}

func relay2Node(nc *nodeClient, data json.RawMessage) {
	enc := json.NewEncoder(nc.c)
	err := enc.Encode(data)
	if err != nil {
		logger.Error(err)
	}
}

func initStratumServer(db *database, conf *config) {
	ln, err := net.Listen("tcp",
		conf.StratumServer.Address+":"+strconv.Itoa(conf.StratumServer.Port),
	)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Warning("listening on ", conf.StratumServer.Port)

	ss := &stratumServer{
		db:   db,
		ln:   ln,
		conf: conf,
	}

	//go ss.backupPerInterval()

	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.Error(err)
		}

		go ss.handleConn(conn)
	}
}

// Deleted
//func (ss *stratumServer) backupPerInterval() {
//	d, err := time.ParseDuration(ss.conf.StratumServer.BackupInterval)
//	if err != nil {
//		logger.Println("failed to start export system", err)
//		return
//	}
//
//	logger.Println("export system running")
//
//	ch := time.Tick(d)
//	for {
//		select {
//		case <-ch:
//			newFileName := strconv.Itoa(time.Now().Year()) + "-" +
//				time.Now().Month().String() + "-" + strconv.Itoa(time.Now().Day()) +
//				"-" + strconv.Itoa(time.Now().Hour())
//			f, err := os.Create(newFileName + ".csv")
//			if err != nil {
//				logger.Println(err)
//				continue
//			}
//			_ = ss.db.View(func(txn *badger.Txn) error {
//				it := txn.NewIterator(badger.DefaultIteratorOptions)
//				defer it.Close()
//				prefix := []byte("shares+")
//				for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
//					item := it.Item()
//					k := item.Key()
//					_ = item.Value(func(v []byte) error {
//						_, err = fmt.Fprintf(f, "%s %d\n", k, new(big.Int).SetBytes(v).Uint64())
//						logger.Println(err)
//						return nil
//					})
//				}
//
//				return nil
//			})
//			_ = f.Close()
//		}
//	}
//}
