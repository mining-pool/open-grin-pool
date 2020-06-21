package main

import (
	"fmt"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("pool")

func main() {
	var conf = parseConfig()

	lvl, err := logging.LevelFromString(conf.Log.Level)
	if err != nil {
		panic(err)
	}
	logging.SetAllLoggers(lvl) // (logCfg)

	if len(conf.Log.File) > 0 {
		fmt.Println("all log will write to", conf.Log.File)
		logCfg := logging.Config{}
		logCfg.File = conf.Log.File
		logging.SetupLogging(logCfg)
	}

	db := initDB(conf)

	go initAPIServer(db, conf)
	go initStratumServer(db, conf)
	go initPayer(db, conf)
	go initUnlocker(db, conf)
	for {
		select {}
	}
}
