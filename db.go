package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

type database struct {
	prefix string

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
		panic(err)
	}

	return &database{config.Storage.Prefix + ":", rdb, config}
}

func (db *database) registerMiner(login, pass, payment string) {
	_, err := db.client.HMSet(db.prefix+"user:"+login, map[string]interface{}{
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
	correctPassword minerLoginStatusCode // 0
	noPassword      minerLoginStatusCode = 1
	wrongPassword   minerLoginStatusCode = 2
)

func (db *database) verifyMiner(login, pass string) minerLoginStatusCode {
	passInDB, err := db.client.HGet(db.prefix+"user:"+login, "pass").Result()
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
	_, err := db.client.HMSet(db.prefix+"user:"+login, map[string]interface{}{
		"payment": payment,
	}).Result()
	if err != nil {
		log.Error(err)
	}
}

func (db *database) putShare(login, agent string, diff int64) {
	db.putDayShare(login, diff)
	db.putTmpShare(login, agent, diff)

	_, err := db.client.HSet(db.prefix+"user:"+login, "lastShare", time.Now().UnixNano()).Result()
	if err != nil {
		log.Error(err)
	}
}

func (db *database) putDayShare(login string, diff int64) {
	_, err := db.client.HIncrBy(db.prefix+"shares", login, diff).Result()
	if err != nil {
		log.Error(err)
	}
}

func (db *database) putTmpShare(login, agent string, diff int64) {
	z := redis.Z{
		Score:  float64(time.Now().UnixNano()),
		Member: strconv.FormatInt(diff, 10) + ":" + strconv.FormatInt(time.Now().UnixNano(), 16),
	}
	_, err := db.client.ZAdd(db.prefix+"tmp:"+login+":"+agent, z).Result()
	if err != nil {
		log.Error(err)
	}
}

func (db *database) getShares() map[string]string {
	shares, err := db.client.HGetAll(db.prefix + "shares").Result()
	if err != nil {
		log.Error(err)
	}

	return shares
}

func (db *database) getMinerStatus(login string) map[string]interface{} {
	m, err := db.client.HGetAll(db.prefix + "user:" + login).Result()
	if err != nil {
		log.Error(err)
	}

	rtn := make(map[string]interface{})
	for k, v := range m {
		if k == "agents" {
			var agents map[string]interface{}
			_ = json.Unmarshal([]byte(v), &agents)

			rtn["agents"] = agents
		} else {
			rtn[k] = v
		}
	}

	monthStartDay := time.Date(time.Now().Year(), time.Now().Month(), 0, 0, 0, 0, 0, time.Now().Location())
	dateStart, _ := strconv.ParseFloat(monthStartDay.Format("20060102"), 10)
	dateEnd, _ := strconv.ParseFloat(time.Now().Format("20060102"), 10)
	dayRevenues, _ := db.client.ZRangeWithScores(db.prefix+"revenue:"+login, int64(dateStart), int64(dateEnd)).Result()
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
	s, err := db.client.HGet(db.prefix+"user:"+login, "agents").Result()
	if err != nil {
		log.Error("failed to get miner agent, redis answering", err, " maybe on initialization?")
	}

	strUnixNano, _ := db.client.HGet(db.prefix+"user:"+login, "lastShare").Result()
	lastShareTime, _ := strconv.ParseInt(strUnixNano, 10, 64)
	// H = D / ΔT
	if time.Now().UnixNano() == lastShareTime {
		realtimeHashrate := float64(diff*1e9) / (1)
		status["realtime_hashrate"] = realtimeHashrate
	} else {
		realtimeHashrate := float64(diff*1e9) / float64(time.Now().UnixNano()-lastShareTime)
		status["realtime_hashrate"] = realtimeHashrate
	}

	db.client.ZRemRangeByScore(db.prefix+"tmp:"+login+":"+agent, "-inf", fmt.Sprint("(", time.Now().UnixNano()-10*time.Minute.Nanoseconds()))
	l, err := db.client.ZRangeWithScores(db.prefix+"tmp:"+login+":"+agent, 0, -1).Result()
	if err != nil {
		log.Error(err)
	}

	var sum int64
	for _, z := range l {
		str := z.Member.(string)
		li := strings.Split(str, ":")
		i, err := strconv.Atoi(li[0])
		if err != nil {
			log.Error(err)
		}
		sum = sum + int64(i)
	}
	averageHashrate := float64(sum*1e9) / float64(10*time.Minute.Nanoseconds())
	status["average_hashrate"] = averageHashrate

	agents := make(map[string]interface{})
	_ = json.Unmarshal([]byte(s), &agents)

	agents[agent] = status

	raw, _ := json.Marshal(agents)
	_, err = db.client.HSet(db.prefix+"user:"+login, "agents", raw).Result()
	if err != nil {
		log.Error(err)
	}
}

func (db *database) putBlockHash(hash string) {
	_, err := db.client.LPush(db.prefix+"blocksFound", hash).Result()
	if err != nil {
		log.Error(err)
	}
}

func (db *database) getAllBlockHashesFrom(pos int64) []string {
	l, err := db.client.LRange(db.prefix+"blocksFound", pos, -1).Result()
	if err != nil {
		log.Error(err)
	}

	return l
}

type MinedBlock struct {
	Height uint64
	Hash   string
}

func (db *database) putMinedBlock(height uint64, hash string) {
	minedBlock := MinedBlock{
		Height: height,
		Hash:   hash,
	}
	raw, _ := json.Marshal(minedBlock)
	_, err := db.client.LPush(db.prefix+"blocksMined", raw).Result()
	if err != nil {
		log.Error(err)
	}
}

func (db *database) getAllMinedBlockHashes() []MinedBlock {
	blocks := make([]MinedBlock, 0)

	l, err := db.client.LRange(db.prefix+"blocksMined", 0, -1).Result()
	if err != nil {
		log.Error(err)
	}

	for i := range l {
		var b MinedBlock
		_ = json.Unmarshal([]byte(l[i]), &b)
		blocks = append(blocks, b)
	}

	return blocks
}

func (db *database) calcRevenueToday(totalRevenue uint64) {
	allMinersStrSharesTable, err := db.client.HGetAll(db.prefix + "shares").Result()
	if err != nil {
		log.Error(err)
	}

	newFileName := strconv.Itoa(time.Now().Year()) + "-" +
		time.Now().Month().String() + "-" + strconv.Itoa(time.Now().Day())
	f, err := os.Create(newFileName + ".csv")
	if err != nil {
		log.Error(err)
	}

	var totalShare uint64
	allMinersSharesTable := make(map[string]uint64)
	for miner, shares := range allMinersStrSharesTable {
		_, _ = fmt.Fprintf(f, "%s %s\n", miner, shares)

		allMinersSharesTable[miner], _ = strconv.ParseUint(shares, 10, 64)
		totalShare = totalShare + allMinersSharesTable[miner]
	}

	// clean the share
	_, err = db.client.HDel(db.prefix + "share").Result()
	if err != nil {
		log.Error(err)
	}
	_, _ = fmt.Fprintf(f, "\n")

	allMinersRevenueTable := make(map[string]interface{})
	for miner, shares := range allMinersSharesTable {
		revenue := shares / totalShare * totalRevenue
		allMinersRevenueTable[miner] = revenue

		payment, _ := db.client.HGet(db.prefix+"user:"+miner, "payment").Result()
		_, _ = fmt.Fprintf(f, "%s %d %s\n", miner, revenue, payment)

		date, _ := strconv.ParseFloat(time.Now().Format("20190102"), 10)
		z := redis.Z{
			Score:  date,
			Member: strconv.FormatUint(revenue, 10) + ":" + strconv.FormatInt(int64(date), 10),
		}
		_, err := db.client.ZAdd(db.prefix+"revenue:"+miner, z).Result()
		if err != nil {
			log.Error(err)
		}
	}

	db.client.HMSet(db.prefix+"lastDayRevenue", allMinersRevenueTable)

	_ = f.Close()
}

func (db *database) getLastDayRevenue() map[string]string {
	allMinersRevenueTable, err := db.client.HGetAll(db.prefix + "lastDayRevenue").Result()
	if err != nil {
		log.Error(err)
	}

	return allMinersRevenueTable
}
