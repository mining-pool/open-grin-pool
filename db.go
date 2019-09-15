package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/logger"
)

type database struct {
	client *redis.Client
}

func initDB(config *config) *database {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Storage.Address + ":" + strconv.Itoa(config.Storage.Port),
		Password: config.Storage.Password,
		DB:       config.Storage.Db,
	})

	_, err := rdb.Ping().Result()
	if err != nil {
		logger.Fatal(err)
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
		logger.Error(err)
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
		logger.Error(err)
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
		logger.Error(err)
	}
}

func (db *database) putShare(login, agent string, diff int64) {
	db.putDayShare(login, diff)

	_, err := db.client.HSet("u+"+login, "lastShare", time.Now().UnixNano()).Result()
	if err != nil {
		logger.Error(err)
	}
}

func (db *database) putDayShare(login string, diff int64) {
	_, err := db.client.HIncrBy("shares", login, diff).Result()
	if err != nil {
		logger.Error(err)
	}
}

func (db *database) getShares() map[string]string {
	shares, err := db.client.HGetAll("shares").Result()
	if err != nil {
		logger.Error(err)
	}

	return shares
}

func (db *database) putMinerStatus(login string, statusTable map[string]interface{}) {
	_, err := db.client.HMSet("u+"+login, statusTable).Result()
	if err != nil {
		logger.Error(err)
	}
}

func (db *database) getMinerStatus(login string) map[string]interface{} {
	m, err := db.client.HGetAll("u+" + login).Result()
	if err != nil {
		logger.Error(err)
	}

	rtn := make(map[string]interface{})
	var totalHS int64

	for k, v := range m {
		if k == "agents" {
			var i interface{}

			_ = json.Unmarshal([]byte(v), &i)
			agents, _ := i.(map[string]interface{})
			for _, status := range agents {
				s := status.(map[string]interface{})
				hs, _ := s["hashrate"].(int64)
				totalHS = totalHS + hs
			}
			rtn[k] = i
		} else {
			rtn[k] = v
		}
	}

	rtn["hashrate"] = totalHS
	delete(rtn, "pass")

	return rtn
}

func (db *database) setMinerAgentStatus(login, agent string, diff int64, status map[string]interface{}) {
	s, err := db.client.HGet("u+"+login, "agents").Result()
	if err != nil {
		logger.Error(err)
	}

	strUnixNano, _ := db.client.HGet("u+"+login, "lastShare").Result()
	lastShareTime, _ := strconv.ParseInt(strUnixNano, 10, 64)
	hashrate := diff * 1e9 / (time.Now().UnixNano() - lastShareTime)
	status["hashrate"] = hashrate

	agents := make(map[string]interface{})
	_ = json.Unmarshal([]byte(s), &agents)

	agents[agent] = status

	raw, _ := json.Marshal(agents)
	_, err = db.client.HSet("u+"+login, "agents", raw).Result()
	if err != nil {
		logger.Error(err)
	}
}

func (db *database) putBlockHash(hash string) {
	_, err := db.client.LPush("blocksFound", hash).Result()
	if err != nil {
		logger.Error(err)
	}
}

func (db *database) getAllBlockHashes() []string {
	l, err := db.client.LRange("blocksFound", 0, -1).Result()
	if err != nil {
		logger.Error(err)
	}

	return l
}

func (db *database) calcRevenueToday(totalRevenue uint64) {
	allMinersStrSharesTable, err := db.client.HGetAll("shares").Result()
	if err != nil {
		logger.Error(err)
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

	// clean the share
	db.client.HDel("share")
	_, err = fmt.Fprintf(f, "\n")

	allMinersRevenueTable := make(map[string]interface{})
	for miner, shares := range allMinersSharesTable {
		allMinersRevenueTable[miner] = shares / totalShare * totalRevenue

		payment := db.client.HGet("u+"+miner, "payment")
		_, err = fmt.Fprintf(f, "%s %d %s\n", miner, allMinersSharesTable[miner], payment)
	}

	db.client.HMSet("lastDayRevenue", allMinersRevenueTable)

	_ = f.Close()
}

func (db *database) getLastDayRevenue() map[string]string {
	allMinersRevenueTable, err := db.client.HGetAll("lastDayRevenue").Result()
	if err != nil {
		logger.Error(err)
	}

	return allMinersRevenueTable
}

func (db *database) putPoolStatus() {

}
