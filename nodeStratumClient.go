package main

import (
	"context"
	"encoding/json"
	"net"

	"github.com/google/logger"
)

type nodeClient struct {
	c   net.Conn
	enc *json.Encoder
	dec *json.Decoder
}

func initNodeStratumClient(conf *config) *nodeClient {
	ip := net.ParseIP(conf.Node.Address)
	raddr := &net.TCPAddr{
		IP:   ip,
		Port: conf.Node.StratumPort,
	}
	conn, err := net.DialTCP("tcp4", nil, raddr)
	if err != nil {
		logger.Error(err)
	}

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)
	nc := &nodeClient{
		c:   conn,
		enc: enc,
		dec: dec,
	}

	return nc
}

// registerHandler will hook the callback function to the tcp conn, and call func when recv
func (nc *nodeClient) registerHandler(ctx context.Context, callback func(sr json.RawMessage)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var sr json.RawMessage

			err := nc.dec.Decode(&sr)
			if err != nil {
				logger.Error(err)
				return
			}

			resp, _ := sr.MarshalJSON()
			logger.Info("Node returns a response: ", string(resp))

			go callback(sr)
		}
	}
}

func (nc *nodeClient) close() {
	_ = nc.c.Close()
}
