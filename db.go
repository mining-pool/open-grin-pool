package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

type database struct {
	client *redis.Client
}

func initDB(config *config) *database {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Storage.Address + strconv.Itoa(config.Storage.Port),
		Password: config.Storage.Password,
		DB:       config.Storage.DB,
	})

	_, err := rdb.Ping().Result()
	if err != nil {
		log.Fatal(err)
	}

	return &database{rdb}
}

func (db *database) registerMiner(login, pass, payment string) {
	_, err := db.client.HMSet("u+"+login, map[string]interface{}{
		"pass":      pass,
		"payment":   payment,
		"lastShare": 0,
	}).Result()
	if err != nil {
		log.Error(err)
	}
}

type minerLoginStatusCode int

var (
	correctPassword minerLoginStatusCode = 0
	noPassword      minerLoginStatusCode = 1
	wrongPassword   minerLoginStatusCode = 2
)

func (db *database) verifyMiner(login, pass string) minerLoginStatusCode {
	passInDB, err := db.client.HGet("u+"+login, "pass").Result()
	if err != nil {
		log.Error(err)
	}

	if passInDB == "" || passInDB == "x" {
		return noPassword
	}

	if passInDB != pass {
		return wrongPassword
	}

	return correctPassword
}

func (db *database) updatePayment(login, payment string) {
	_, err := db.client.HMSet("u+"+login, map[string]interface{}{
		"payment": payment,
	}).Result()
	if err != nil {
		log.Error(err)
	}
}

func (db *database) writeShare(login string, diff int64) {
	tx := db.client.TxPipeline()
	tx.HIncrBy("shares", login, diff)
	tx.HSet("u+"+login, "lastShare", time.Now().UnixNano())
	_, err := tx.Exec()
	if err != nil {
		log.Error(err)
	}
}

func (db *database) insertMinerStatus(login string, statusTable map[string]interface{}) {
	_, err := db.client.HMSet("u+"+login, statusTable).Result()
	if err != nil {
		log.Error(err)
	}
}

func (db *database) getMinerStatus(login string) map[string]string {
	m, err := db.client.HGetAll("u+" + login).Result()
	if err != nil {
		log.Error(err)
	}

	return m
}

func (db *database) setMinerStatus(login string, more map[string]interface{}) {
	_, err := db.client.HMSet("u+"+login, more).Result()
	if err != nil {
		log.Error(err)
	}
}

func (db *database) insertBlockHash(hash string) {
	_, err := db.client.LPush("blocksFound", hash).Result()
	if err != nil {
		log.Error(err)
	}
}

func (db *database) getAllBlockHashes() []string {
	l, err := db.client.LRange("blocksFound", 0, -1).Result()
	if err != nil {
		log.Error(err)
	}

	return l
}

func (db *database) calcTodayRevenue(totalRevenue uint64) {
	allMinersStrSharesTable, err := db.client.HGetAll("shares").Result()
	if err != nil {
		log.Error(err)
	}

	newFileName := strconv.Itoa(time.Now().Year()) + "-" +
		time.Now().Month().String() + "-" + strconv.Itoa(time.Now().Day())
	f, err := os.Create(newFileName + ".csv")

	var totalShare uint64
	allMinersSharesTable := make(map[string]uint64)
	for miner, shares := range allMinersStrSharesTable {
		_, err = fmt.Fprintf(f, "%s %d\n", miner, shares)

		allMinersSharesTable[miner], _ = strconv.ParseUint(shares, 10, 64)
		totalShare = totalShare + allMinersSharesTable[miner]
	}

	_, err = fmt.Fprintf(f, "\n")

	allMinersRevenueTable := make(map[string]interface{})
	for miner, shares := range allMinersSharesTable {
		allMinersRevenueTable[miner] = shares / totalShare * totalRevenue

		_, err = fmt.Fprintf(f, "%s %d\n", miner, allMinersSharesTable[miner])
	}

	db.client.HMSet("lastDayRevenue", allMinersRevenueTable)

	_ = f.Close()
}

func (db *database) getLastDayRevenue() map[string]string {
	allMinersRevenueTable, err := db.client.HGetAll("lastDayRevenue").Result()
	if err != nil {
		log.Error(err)
	}

	return allMinersRevenueTable
}

func (db *database) putPoolStatus() {

}
