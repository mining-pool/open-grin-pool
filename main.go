package main

import (
	"os"

	"github.com/google/logger"
)

var conf = parseConfig()
var logFile, _ = os.Create(conf.Log.LogFile)
var log = logger.Init("pool", conf.Log.Verbose, conf.Log.SystemLog, logFile)

func main() {
	db := initDB(conf)

	p := &payer{}
	p.watch()

	go initAPIServer(db, conf)
	go initStratumServer(db, conf)
	for {
		select {}
	}
}
