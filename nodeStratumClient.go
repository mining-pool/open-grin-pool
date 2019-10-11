package main

import (
	"bufio"
	"context"
	"io"
	"net"
	"sync"

	"github.com/google/logger"
	jsoniter "github.com/json-iterator/go"
)

type nodeClient struct {
	conf *config
	conn net.Conn
	rw   *bufio.ReadWriter

	mu sync.Mutex
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

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	nc := &nodeClient{
		conf: conf,
		conn: conn,
		rw:   rw,
	}

	return nc
}

// registerHandler will hook the callback function to the tcp conn, and call func when recv
func (nc *nodeClient) registerHandler(ctx context.Context, callback func(sr jsoniter.RawMessage)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var sr jsoniter.RawMessage

			//err := nc.dec.Decode(&sr)
			raw, err := nc.rw.ReadBytes('\n')
			if err != nil {
				logger.Error(err)
				if err == io.EOF {
					if nc.reconnect() != nil {
						return
					}
				}
				continue
			}

			_ = json.Unmarshal(raw, &sr)
			logger.Info("Node returns a response: ", string(sr))

			go callback(sr)
		}
	}
}

func (nc *nodeClient) Send(req interface{}) {
	raw, err := json.Marshal(req)
	if err != nil {
		logger.Error(err)
	}
	_, err = nc.rw.Write(append(raw, '\n'))
	if err != nil {
		logger.Error(err)
	}
	err = nc.rw.Flush()
	if err != nil {
		logger.Error(err)
	}
}

func (nc *nodeClient) Encode(raw jsoniter.RawMessage) {
	_, err := nc.rw.Write(append(raw, '\n'))
	if err != nil {
		logger.Error(err)
	}
	err = nc.rw.Flush()
	if err != nil {
		logger.Error(err)
	}
}

func (nc *nodeClient) reconnect() error {
	nc.mu.Lock()
	defer nc.mu.Unlock()

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

	nc.conn = conn
	nc.rw = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	return nil
}

func (nc *nodeClient) close() {
	_ = nc.conn.Close()
}
