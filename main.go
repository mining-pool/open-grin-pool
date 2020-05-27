package main

import (
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("pool")

func main() {
	var conf = parseConfig()

	lvl, err := logging.LevelFromString(conf.Log.Level)
	if err != nil {
		panic(err)
	}

	logCfg := logging.Config{
		Level: lvl,
		File:  conf.Log.File,
	}

	logging.SetupLogging(logCfg)

	db := initDB(conf)

	go initAPIServer(db, conf)
	go initStratumServer(db, conf)
	go initPayer(db, conf)
	go initUnlocker(db, conf)
	for {
		select {}
	}
}
