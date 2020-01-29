package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/logger"
)

type payer struct {
	db   *database
	conf *config
}

type jsonRPCResponse struct {
	ID      string                 `json:"id"`
	JsonRpc string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Result  interface{}            `json:"result"`
	Error   map[string]interface{} `json:"error"`
}

// deprecated v1 wallet owner api
func (p *payer) getNewBalanceV1() uint64 {
	req, _ := http.NewRequest("GET", "http://"+p.conf.Wallet.Address+":"+strconv.Itoa(p.conf.Wallet.OwnerAPIPort)+"/v1/wallet/owner/retrieve_summary_info?refresh", nil)
	req.SetBasicAuth(p.conf.Node.AuthUser, p.conf.Node.AuthPass)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		logger.Error("failed to get balance from wallet, treat this as no income")
		return 0
	}

	dec := json.NewDecoder(res.Body)
	var summaryInfo []interface{}
	_ = dec.Decode(&summaryInfo)

	i, _ := summaryInfo[1].(map[string]interface{})
	strSpendable := i["amount_currently_spendable"].(string)
	spendable, _ := strconv.Atoi(strSpendable)
	return uint64(spendable) // unit nanogrin
}

// The V2 Owner API (OwnerRpc) will be removed in grin-wallet 4.0.0. Please migrate to the V3 (OwnerRpcS) API as soon as possible.
func (p *payer) getNewBalanceV2() uint64 {
	req, _ := http.NewRequest("POST", "http://"+p.conf.Wallet.Address+":"+strconv.Itoa(p.conf.Wallet.OwnerAPIPort)+"/v2/wallet/owner", strings.NewReader(`{
	"jsonrpc": "2.0",
	"method": "retrieve_summary_info",
	"params": [true, 10],
	"id": 1
}`))
	req.SetBasicAuth(p.conf.Node.AuthUser, p.conf.Node.AuthPass)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		logger.Error("failed to get balance from wallet for failing to post request, treat this as no income")
		return 0
	}

	dec := json.NewDecoder(res.Body)
	var summaryInfo jsonRPCResponse
	_ = dec.Decode(&summaryInfo)

	result := summaryInfo.Result
	if result == nil {
		logger.Error(summaryInfo.Error)
		logger.Error("failed to get balance from wallet for failing to get result request, treat this as no income")
		return 0
	}

	mapResult, ok := result.(map[string]interface{})
	if ok || mapResult["Ok"] != nil {
		theOk, ok := mapResult["Ok"].([]interface{})
		if ok {
			for i := range theOk {
				if balanceMap, ok := theOk[i].(map[string]interface{}); ok {
					strSpendable := balanceMap["amount_currently_spendable"].(string)
					spendable, _ := strconv.Atoi(strSpendable)
					return uint64(spendable) // unit nanogrin
				}
			}
		}
	}

	return 0
}

func (p *payer) getNewBalanceV3() uint64 {
	logger.Fatal("V3 support WIP!")
	return 0
}

// distribute coins when balance is > 1e9 nano
func (p *payer) distribute(newBalance uint64) {
	// get a distribution table
	revenue4Miners := uint64(float64(newBalance) * (1 - p.conf.Payer.Fee))
	p.db.calcRevenueToday(revenue4Miners)
}

func (p *payer) watch() {
	go func() {
		m := strings.Split(p.conf.Payer.Time, ":")
		hour, err := strconv.Atoi(m[0])
		if err != nil {
			logger.Error(err)
		}
		min, err := strconv.Atoi(m[1])
		if err != nil {
			logger.Error(err)
		}

		var getNewBalance func() uint64
		switch p.conf.Wallet.OwnerAPIVersion {
		case "v1":
			getNewBalance = p.getNewBalanceV1
		case "v2":
			getNewBalance = p.getNewBalanceV2
		case "v3":
			getNewBalance = p.getNewBalanceV3
		}

		for {
			now := time.Now()
			t := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, now.Location())
			if t.After(now) == false {
				next := now.Add(time.Hour * 24)
				t = time.Date(next.Year(), next.Month(), next.Day(), hour, min, 0, 0, next.Location())
			}
			timer := time.NewTimer(t.Sub(now))

			select {
			case <-timer.C:
				newBalance := getNewBalance()
				if newBalance > 1e9 {
					p.distribute(newBalance - 1e9)
				} else {
					p.distribute(0)
				}
			}
		}
	}()
}

func initPayer(db *database, conf *config) *payer {
	p := &payer{
		db:   db,
		conf: conf,
	}
	p.watch()

	return p
}
