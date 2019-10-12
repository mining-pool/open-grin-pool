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
	dec  *jsoniter.Decoder
	w    *bufio.Writer
	mu   sync.Mutex
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

	dec := json.NewDecoder(conn)
	w := bufio.NewWriter(conn)
	nc := &nodeClient{
		conf: conf,
		conn: conn,
		w:    w,
		dec:  dec,
	}

	return nc
}

// registerHandler will hook the callback function to the tcp conn, and call func when recv
func (nc *nodeClient) registerHandler(ctx context.Context, callback func(jRpc JsonRPC)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var jRpc JsonRPC
			err := nc.dec.Decode(&jRpc)
			if err != nil {
				if err == io.EOF {
					continue
				} else {
					if nc.reconnect() != nil {
						return
					}
				}
				continue
			}

			logger.Info("Node returns a response: ", jRpc.String())

			go callback(jRpc)
		}
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
	nc.dec = json.NewDecoder(conn)
	nc.w = bufio.NewWriter(conn)
	return nil
}

func (nc *nodeClient) Close() {
	_ = nc.conn.Close()
}

func (nc *nodeClient) Encode(msg interface{}) {
	raw, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err)
	}
	_, err = nc.w.Write(raw)
	if err != nil {
		logger.Error(err)
	}
	err = nc.w.Flush()
	if err != nil {
		logger.Error(err)
	}
}
