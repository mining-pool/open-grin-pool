package main

import (
	"os"

	"github.com/google/logger"
)

func main() {
	var conf = parseConfig()
	lf, err := os.OpenFile(conf.Log.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		logger.Errorf("Failed to open log file: %v", err)
	}
	defer lf.Close()
	defer logger.Init("pool", conf.Log.Verbose, conf.Log.SystemLog, lf).Close()

	db := initDB(conf)

	go initAPIServer(db, conf)
	go initStratumServer(db, conf)
	go initPayer(db, conf)
	go initUnlocker(db, conf)
	for {
		select {}
	}
}
