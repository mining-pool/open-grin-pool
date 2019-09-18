package main

// http rpc server
import (
	"context"
	"encoding/json"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"syscall"
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
	agent      string
	difficulty int64
	ctx        context.Context
}

func (ms *minerSession) hasNotLoggedIn() bool {
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
		db.setMinerAgentStatus(ms.login, ms.agent, ms.difficulty, result)

		break
	case "submit":
		db.putShare(ms.login, ms.agent, ms.difficulty)
		if res.Error != nil {
			logger.Warning(ms.login, "'s share has err: ", res.Error)
			break
		}
		detail, ok := res.Result.(string)
		logger.Info(ms.login, " has submit a ", detail, " share")
		if ok {
			if strings.Contains(detail, "block - ") {
				blockHash := strings.Trim(detail, "block - ")
				db.putBlockHash(blockHash)
				logger.Warning("block ", blockHash, " has been found by ", ms.login)
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
	session := &minerSession{difficulty: int64(ss.conf.Node.Diff)}
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
			opErr, ok := err.(*net.OpError)
			if ok {
				if opErr.Err.Error() == syscall.ECONNRESET.Error() {
					return
				}
			} else {
				logger.Error(err)
			}
		}

		err = json.Unmarshal(jsonRaw, &clientReq)
		if err != nil {
			// logger.Error(err)
			continue
		}

		logger.Info(conn.RemoteAddr(), " sends a ", clientReq.Method, " request:", string(jsonRaw))

		switch clientReq.Method {
		case "login":
			login, _ = clientReq.Params["login"].(string)

			pass, _ := clientReq.Params["pass"].(string)

			agent, _ := clientReq.Params["agent"].(string)

			login = strings.TrimSpace(login)
			pass = strings.TrimSpace(pass)
			agent = strings.TrimSpace(agent)

			if agent == "" {
				agent = "NoNameMiner" + strconv.FormatInt(rand.Int63(), 10)
			}

			switch ss.db.verifyMiner(login, pass) {
			case wrongPassword:
				logger.Warning(login, " has failed to login")
				login = ""
				_, _ = conn.Write([]byte(`{  
   "id":"5",
   "jsonrpc":"2.0",
   "method":"login",
   "error":{  
      "code":-32500,
      "message":"login incorrect"
   }
}`))

			case noPassword:
				ss.db.registerMiner(login, pass, "")
				logger.Info(login, " has registered in")

			case correctPassword:

			}

			session.login = login
			session.agent = agent
			logger.Info(session.login, "'s ", agent, " has logged in")
			go relay2Node(nc, jsonRaw)

		default:
			if session.hasNotLoggedIn() {
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
		opErr, ok := err.(*net.OpError)
		if ok {
			if opErr.Err.Error() == "use of closed network connection" {
				return
			}
		} else {
			logger.Error(err)
		}
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
