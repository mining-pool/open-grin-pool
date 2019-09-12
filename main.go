package main

import "github.com/op/go-logging"

var log = logging.MustGetLogger("pool")

func main() {
	conf := parseConfig()
	db := initDB(conf)

	p := &payer{}
	p.watch()

	go initAPIServer(db, conf)
	go initStratumServer(db, conf)
	for {
		select {}
	}
}
