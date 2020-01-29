package main

import (
	"encoding/json"
	"github.com/google/logger"
	"net/http"
	"strconv"
	"strings"
)

// https://docs.rs/grin_wallet_api/3.0.0/grin_wallet_api
type OwnerAPI struct {
	db   *database
	conf *config
}

func NewOwnerAPI(db *database, conf *config) *OwnerAPI {
	return &OwnerAPI{
		db:   db,
		conf: conf,
	}
}

// deprecated v1 wallet owner api
func (o *OwnerAPI) getNewBalanceV1() uint64 {
	req, _ := http.NewRequest("GET", "http://"+o.conf.Wallet.Address+":"+strconv.Itoa(o.conf.Wallet.OwnerAPIPort)+"/v1/wallet/owner/retrieve_summary_info?refresh", nil)
	req.SetBasicAuth(o.conf.Node.AuthUser, o.conf.Node.AuthPass)
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
func (o *OwnerAPI) getNewBalanceV2() uint64 {
	req, _ := http.NewRequest("POST", "http://"+o.conf.Wallet.Address+":"+strconv.Itoa(o.conf.Wallet.OwnerAPIPort)+"/v2/wallet/owner", strings.NewReader(`{
	"jsonrpc": "2.0",
	"method": "retrieve_summary_info",
	"params": [true, 10],
	"id": 1
}`))
	req.SetBasicAuth(o.conf.Node.AuthUser, o.conf.Node.AuthPass)
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

func (o *OwnerAPI) getNewBalanceV3() uint64 {
	logger.Fatal("V3 support WIP!")
	return 0
}
