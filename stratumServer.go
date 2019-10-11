package main

// http rpc server
import (
	"bufio"
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

type stratumRequest struct {
	ID      string                 `json:"id"`
	JsonRpc string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
}

type JsonRPC struct {
	ID      string                 `json:"id"`
	JsonRpc string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Result  interface{}            `json:"result, omitempty"`
	Params  map[string]interface{} `json:"params, omitempty"`
	Error   map[string]interface{} `json:"error, omitempty"`
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

	conf *config
	ctx  context.Context
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
	statusReq := &stratumRequest{
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

	go nc.registerHandler(ctx, func(sr jsoniter.RawMessage) {
		w := bufio.NewWriter(conn)
		b, err := json.Marshal(sr)
		if err != nil {
			logger.Error(err)
		}

		var msg JsonRPC
		_ = json.Unmarshal(sr, &msg) // suppress the err
		if msg.Method == "job" {
			jobAlgo := msg.Params["algorithm"]
			if ss.conf.StratumServer[ss.id].Algo != jobAlgo {
				return
			}
		}

		_, err = w.Write(append(b, '\n'))
		err = w.Flush()
		if err != nil {
			logger.Error(err)
		}

		if err != nil {
			logger.Error(err)
		}

		// internal record
		var res stratumResponse
		_ = json.Unmarshal(sr, &res)
		session.handleMethod(&res, ss.db)
	})
	defer nc.close()

	for {
		var clientReq stratumRequest

		raw, err := nc.rw.ReadBytes('\n')
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

		err = json.Unmarshal(raw, &clientReq)
		if err != nil {
			// logger.Error(err)
			continue
		}

		logger.Info(conn.RemoteAddr(), " sends a ", clientReq.Method, " request:", string(raw))

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
			nc.Encode(raw)

		default:
			if session.hasNotLoggedIn() {
				logger.Warning(login, " has not logged in")
			}

			nc.Encode(raw)
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
