package main

import (
	"context"
	"encoding/json"
	"net"
	"strconv"

	"github.com/google/logger"
)

type nodeClient struct {
	c net.Conn
}

func initNodeStratumClient(conf *config) *nodeClient {
	conn, err := net.Dial("tcp4", conf.Node.Address+":"+strconv.Itoa(conf.Node.StratumPort))
	if err != nil {
		logger.Error(err)
	}

	nc := &nodeClient{
		c: conn,
	}

	return nc
}

// registerHandler will hook the callback function to the tcp conn, and call func when recv
func (nc *nodeClient) registerHandler(ctx context.Context, callback func(sr json.RawMessage)) {
	defer nc.c.Close()
	dec := json.NewDecoder(nc.c)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var sr json.RawMessage

			err := dec.Decode(&sr)
			if err != nil {
				logger.Error(err)
				return
			}

			resp, _ := sr.MarshalJSON()
			logger.Info("Node returns a response: ", resp)

			go callback(sr)
		}
	}
}

func (nc *nodeClient) close() {
	_ = nc.c.Close()
}
