package main

import (
	"testing"
)

func TestPutTmpShare(t *testing.T) {
	//conf := parseConfig()
	//db := initDB(conf)
	//login := "123"
	//agent := "321"
	//ch := time.After(10 * time.Minute)
	//for {
	//	select {
	//	case <-ch:
	//		goto SUM
	//	default:
	//		time.Sleep(10*time.Second)
	//		db.putTmpShare(login, agent, 5000)
	//	}
	//}
	//
	//SUM:
	//
	//db.client.ZRemRangeByScore("tmp:"+login+":"+agent, "-inf", fmt.Sprint("(", time.Now().UnixNano()-10*time.Minute.Nanoseconds()))
	//l, err := db.client.ZRangeWithScores("tmp:"+login+":"+agent, 0, -1).Result()
	//if err != nil {
	//	logger.Error(err)
	//}
	//
	//logger.Info(l)
	//
	//var sum int64
	//for _, z := range l {
	//	str := z.Member.(string)
	//	li := strings.Split(str, ":")
	//	logger.Info(li)
	//	i, err := strconv.Atoi(li[0])
	//	if err != nil {
	//		logger.Error(err)
	//	}
	//	sum = sum + int64(i)
	//}
	//logger.Info(sum)
	//logger.Info(time.Minute.Seconds())
	//logger.Info(int64(db.conf.Node.BlockTime))
	////H = D / ΔT
	//averageHashrate := sum / (10 * int64(time.Minute.Seconds()))
	//logger.Info(averageHashrate)
}
