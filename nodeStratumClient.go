package main

import (
	"encoding/json"
	"log"
	"net"
	"strconv"
)

type nodeClient struct {
	c net.Conn
}

func initNodeStratumClient(conf *config) *nodeClient {
	conn, err := net.Dial("tcp4", conf.Node.Address+":"+strconv.Itoa(conf.Node.StratumPort))
	if err != nil {
		log.Panic(err)
	}

	nc := &nodeClient{
		c: conn,
	}

	return nc
}

func (nc *nodeClient) wait(callback func(sr json.RawMessage)) {
	defer nc.c.Close()
	dec := json.NewDecoder(nc.c)

	for {
		var sr json.RawMessage

		err := dec.Decode(&sr)
		if err != nil {
			log.Println(err)
			return
		}
		callback(sr)
	}
}
