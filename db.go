package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/logger"
)

type database struct {
	client *redis.Client
	conf   *config
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

	return &database{rdb, config}
}

func (db *database) registerMiner(login, pass, payment string) {
	_, err := db.client.HMSet("user:"+login, map[string]interface{}{
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
	passInDB, err := db.client.HGet("user:"+login, "pass").Result()
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
	_, err := db.client.HMSet("user:"+login, map[string]interface{}{
		"payment": payment,
	}).Result()
	if err != nil {
		logger.Error(err)
	}
}

func (db *database) putShare(login, agent string, diff int64) {
	db.putDayShare(login, diff)
	db.putTmpShare(login, agent, diff)

	_, err := db.client.HSet("user:"+login, "lastShare", time.Now().UnixNano()).Result()
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

func (db *database) putTmpShare(login, agent string, diff int64) {
	z := redis.Z{
		Score:  float64(time.Now().UnixNano()),
		Member: strconv.FormatInt(diff, 10) + ":" + strconv.FormatInt(time.Now().Unix(), 10),
	}
	err := db.client.ZAdd("tmp:"+login+":"+agent, z)
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
	_, err := db.client.HMSet("user:"+login, statusTable).Result()
	if err != nil {
		logger.Error(err)
	}
}

func (db *database) getMinerStatus(login string) map[string]interface{} {
	m, err := db.client.HGetAll("user:" + login).Result()
	if err != nil {
		logger.Error(err)
	}

	rtn := make(map[string]interface{})
	var totalHS int64

	for k, v := range m {
		if k == "agents" {
			var agents map[string]interface{}
			_ = json.Unmarshal([]byte(v), &agents)

			for _, agentStatus := range agents {
				s := agentStatus.(map[string]interface{})
				hs, _ := s["average_hashrate"].(int64)
				totalHS = totalHS + hs
			}
			rtn["agents"] = agents
		} else {
			rtn[k] = v
		}
	}

	rtn["hashrate"] = totalHS

	monthStartDay := time.Date(time.Now().Year(), time.Now().Month(), 0, 0, 0, 0, 0, time.Now().Location())
	dateStart, _ := strconv.ParseFloat(monthStartDay.Format("20190102"), 10)
	dateEnd, _ := strconv.ParseFloat(time.Now().Format("20190102"), 10)
	dayRevenues, _ := db.client.ZRangeWithScores("revenue:"+login, int64(dateStart), int64(dateEnd)).Result()
	table := make(map[string]interface{})
	for _, z := range dayRevenues {
		str, _ := z.Member.(string)
		li := strings.Split(str, ":")
		if len(li) < 2 {
			continue
		}
		table[strconv.FormatInt(int64(z.Score), 10)] = li[0]
	}

	rtn["revenues"] = table

	delete(rtn, "pass")

	return rtn
}

func (db *database) setMinerAgentStatus(login, agent string, diff int64, status map[string]interface{}) {
	s, err := db.client.HGet("user:"+login, "agents").Result()
	if err != nil {
		logger.Error(err)
	}

	strUnixNano, _ := db.client.HGet("user:"+login, "lastShare").Result()
	lastShareTime, _ := strconv.ParseInt(strUnixNano, 10, 64)
	realtimeHashrate := diff * 1e9 / (time.Now().UnixNano() - lastShareTime) / int64(db.conf.Node.BlockTime)
	status["realtime_hashrate"] = realtimeHashrate

	l, _ := db.client.ZRange("tmp:"+login+":"+agent, time.Now().UnixNano()-10*time.Minute.Nanoseconds(), time.Now().UnixNano()).Result()
	var sum int64
	for _, str := range l {
		li := strings.Split(str, ":")
		if len(li) < 2 {
			continue
		}
		i, err := strconv.Atoi(li[0])
		if err != nil {
			logger.Error(err)
		}
		sum = sum + int64(i)
	}
	averageHashrate := sum * 1e9 / (10 * time.Minute.Nanoseconds()) / int64(db.conf.Node.BlockTime)
	status["average_hashrate"] = averageHashrate

	agents := make(map[string]interface{})
	_ = json.Unmarshal([]byte(s), &agents)

	agents[agent] = status

	raw, _ := json.Marshal(agents)
	_, err = db.client.HSet("user:"+login, "agents", raw).Result()
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
		_, err = fmt.Fprintf(f, "%s %s\n", miner, shares)

		allMinersSharesTable[miner], _ = strconv.ParseUint(shares, 10, 64)
		totalShare = totalShare + allMinersSharesTable[miner]
	}

	// clean the share
	db.client.HDel("share")
	_, err = fmt.Fprintf(f, "\n")

	allMinersRevenueTable := make(map[string]interface{})
	for miner, shares := range allMinersSharesTable {
		allMinersRevenueTable[miner] = shares / totalShare * totalRevenue

		payment := db.client.HGet("user:"+miner, "payment")
		_, err = fmt.Fprintf(f, "%s %d %s\n", miner, allMinersSharesTable[miner], payment)

		date, _ := strconv.ParseFloat(time.Now().Format("20190102"), 10)
		z := redis.Z{
			Score:  date,
			Member: strconv.FormatUint(allMinersSharesTable[miner], 10) + ":" + strconv.FormatInt(int64(date), 10),
		}
		db.client.ZAdd("revenue:"+miner, z)
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
