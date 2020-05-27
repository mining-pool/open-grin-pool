package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type BlockUnlocker struct {
	db   *database
	conf *config

	lastMinedInFoundPos int64
}

func NewBlockUnlocker(db *database, conf *config) *BlockUnlocker {
	return &BlockUnlocker{
		db:                  db,
		conf:                conf,
		lastMinedInFoundPos: 0,
	}
}

func (u *BlockUnlocker) readLatestFoundHashes() []string {
	return u.db.getAllBlockHashesFrom(u.lastMinedInFoundPos)
}

func (u *BlockUnlocker) checkMature(hash string) int64 {
	req, _ := http.NewRequest("GET", "http://"+u.conf.Node.Address+":"+strconv.Itoa(u.conf.Node.APIPort)+"/v1/blocks/"+hash, nil)
	req.SetBasicAuth(u.conf.Node.AuthUser, u.conf.Node.AuthPass)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Error(err)
	}

	if res.StatusCode == 200 {
		content := make(map[string]interface{})
		dec := json.NewDecoder(res.Body)
		if err := dec.Decode(&content); err == nil || content["header"] != nil {
			if headerMap, ok := content["header"].(map[string]interface{}); ok || headerMap["height"] != nil {
				return headerMap["height"].(int64)
			}
		}
	}

	return -1
}

func (u *BlockUnlocker) watch() {
	timeIntervalCh := time.Tick(time.Minute * 5)
	for {
		select {
		case <-timeIntervalCh:
			for _, hash := range u.readLatestFoundHashes() {
				if h := u.checkMature(hash); h >= 0 {
					u.db.putMinedBlock(uint64(h), hash)
				}
			}
		}
	}
}

func initUnlocker(db *database, conf *config) {
	unlocker := NewBlockUnlocker(db, conf)
	unlocker.watch()
}
