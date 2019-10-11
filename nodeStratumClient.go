package main

import (
	"bufio"
	"context"
	"net"
	"sync"

	"github.com/google/logger"
)

type nodeClient struct {
	conf *config
	conn net.Conn
	w    *bufio.Writer
	s    *bufio.Scanner
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

	w := bufio.NewWriter(conn)
	s := bufio.NewScanner(bufio.NewReader(conn))

	nc := &nodeClient{
		conf: conf,
		conn: conn,
		w:    w,
		s:    s,
	}

	return nc
}

// registerHandler will hook the callback function to the tcp conn, and call func when recv
func (nc *nodeClient) registerHandler(ctx context.Context, callback func(raw []byte)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			raw := nc.s.Bytes()
			//if err != nil {
			//	if err == io.EOF {
			//		continue
			//	} else {
			//		if nc.reconnect() != nil {
			//			return
			//		}
			//	}
			//	continue
			//}
			logger.Info("Node returns a response: ", string(raw))

			go callback(raw)
			if nc.s.Err() != nil {
				logger.Error(nc.s.Err())
				if nc.reconnect() != nil {
					return
				}
				continue
			}

		}
	}
}

func (nc *nodeClient) Send(req interface{}) {
	raw, err := json.Marshal(req)
	if err != nil {
		logger.Error(err)
	}
	_, err = nc.w.Write(append(raw, '\n'))
	if err != nil {
		logger.Error(err)
	}
	err = nc.w.Flush()
	if err != nil {
		logger.Error(err)
	}
}

func (nc *nodeClient) Write(raw []byte) {
	_, err := nc.w.Write(append(raw, '\n'))
	if err != nil {
		logger.Error(err)
	}
	err = nc.w.Flush()
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
	nc.w = bufio.NewWriter(conn)
	nc.s = bufio.NewScanner(conn)

	return nil
}

func (nc *nodeClient) close() {
	_ = nc.conn.Close()
}
