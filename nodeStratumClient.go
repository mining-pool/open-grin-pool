package main

import (
	"context"
	"encoding/json"
	"io"
	"net"

	"github.com/google/logger"
)

type nodeClient struct {
	conf *config
	c    net.Conn
	enc  *json.Encoder
	dec  *json.Decoder
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
		conf: conf,
		c:    conn,
		enc:  enc,
		dec:  dec,
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
				if err == io.EOF {
					if nc.reconnect() != nil {
						return
					}
				}
				continue
			}

			resp, _ := sr.MarshalJSON()
			logger.Info("Node returns a response: ", string(resp))

			go callback(sr)
		}
	}
}

func (nc *nodeClient) reconnect() error {
	ip := net.ParseIP(nc.conf.Node.Address)
	raddr := &net.TCPAddr{
		IP:   ip,
		Port: nc.conf.Node.StratumPort,
	}
	conn, err := net.DialTCP("tcp4", nil, raddr)
	if err != nil {
		logger.Error(err)
		return err
	}

	nc.c = conn
	nc.enc = json.NewEncoder(conn)
	nc.dec = json.NewDecoder(conn)

	return nil
}

func (nc *nodeClient) close() {
	_ = nc.c.Close()
}
