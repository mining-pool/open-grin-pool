package main

// http rpc server
import (
	"context"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/logger"
	"github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type stratumServer struct {
	id   int
	db   *database
	ln   net.Listener
	conf *config
}

type JsonRPC struct {
	ID      string      `json:"id"`
	JsonRpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Result  interface{} `json:"result, omitempty"`
	Params  interface{} `json:"params, omitempty"`
	Error   interface{} `json:"error, omitempty"`
}

func (j JsonRPC) String() string {
	b, _ := json.Marshal(j)
	return string(b)
}

type minerSession struct {
	login      string
	agent      string
	difficulty int64

	conf *config
	ctx  context.Context
}

func (ms *minerSession) hasNotLoggedIn() bool {
	return ms.login == ""
}

func (ms *minerSession) handleMethod(res JsonRPC, db *database) {
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
		if res.Error != nil {
			logger.Warning(ms.login, "'s share has err: ", res.Error)
			break
		}
		detail, ok := res.Result.(string)
		logger.Info(ms.login, " has submit a ", detail, " share")
		if ok {
			db.putShare(ms.login, ms.agent, ms.difficulty)
			if strings.Contains(detail, "block") {
				blockHash := strings.Trim(detail, "block - ")
				db.putBlockHash(blockHash)
				logger.Warning("block ", blockHash, " has been found by ", ms.login)
			}
		}
		break
	}
}

func callStatusPerInterval(ctx context.Context, nc *nodeClient) {
	statusReq := JsonRPC{
		ID:      "0",
		JsonRpc: "2.0",
		Method:  "status",
		Params:  nil,
	}

	ch := time.Tick(10 * time.Second)

	for {
		select {
		case <-ch:
			nc.Send(statusReq)
		case <-ctx.Done():
			return
		}
	}
}

func (ss *stratumServer) handleConn(conn net.Conn) {
	logger.Info("new conn from ", conn.RemoteAddr())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session := &minerSession{
		difficulty: int64(ss.conf.Node.Diff),
		conf:       ss.conf,
		ctx:        ctx,
	}

	defer conn.Close()
	var login string
	nc := initNodeStratumClient(ss.conf)

	go callStatusPerInterval(ctx, nc)

	go nc.registerHandler(ctx, func(jRpc JsonRPC) {
		enc := json.NewEncoder(conn)
		if jRpc.Method == "job" {
			if params, ok := jRpc.Params.(map[string]interface{}); ok {
				if jobAlgo, ok := params["algorithm"].(string); ok && ss.conf.StratumServer[ss.id].Algo != jobAlgo {
					return
				}
			}
		}

		err := enc.Encode(&jRpc)
		if err != nil {
			logger.Error(err)
		}

		if err != nil {
			logger.Error(err)
		}

		// internal record
		session.handleMethod(jRpc, ss.db)
	})
	defer nc.Close()

	var dec = json.NewDecoder(conn)
	for {
		var clientReq JsonRPC

		err := dec.Decode(&clientReq)
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

		logger.Info(conn.RemoteAddr(), " sends a ", clientReq.Method, " request:", clientReq.String())

		switch clientReq.Method {
		case "login":
			var ok bool
			params, ok := clientReq.Params.(map[string]interface{})
			if !ok {
				logger.Error("Failed to parse params", clientReq.String())
			}
			login, ok = params["login"].(string)
			if !ok {
				logger.Error("failed to parse login")
			}
			pass, _ := params["pass"].(string)
			if !ok {
				logger.Error("failed to parse pass")
			}
			agent, _ := params["agent"].(string)
			if !ok {
				logger.Error("failed to parse agent")
			}

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
			_ = nc.enc.Encode(&clientReq)

		default:
			if session.hasNotLoggedIn() {
				logger.Warning(login, " has not logged in")
			}

			_ = nc.enc.Encode(&clientReq)
		}
	}
}

func initStratumServer(id int, db *database, conf *config) {
	ip := net.ParseIP(conf.StratumServer[id].Address)
	addr := &net.TCPAddr{
		IP:   ip,
		Port: conf.StratumServer[id].Port,
	}
	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Warning("listening on ", conf.StratumServer[id].Port)

	ss := &stratumServer{
		id:   id,
		db:   db,
		ln:   ln,
		conf: conf,
	}

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			logger.Error(err)
		}

		go ss.handleConn(conn)
	}
}
