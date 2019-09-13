package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type payer struct {
	db   *database
	conf *config
}

func (p *payer) getNewBalance() uint64 {
	req, _ := http.NewRequest("GET", "http://"+p.conf.Wallet.Address+":"+strconv.Itoa(p.conf.Wallet.OwnerAPIPort)+"/v1/wallet/owner/retrieve_summary_info?refresh", nil)
	req.SetBasicAuth(p.conf.Node.AuthUser, p.conf.Node.AuthPass)
	client := &http.Client{}
	res, _ := client.Do(req)

	dec := json.NewDecoder(res.Body)
	var summaryInfo []interface{}
	_ = dec.Decode(&summaryInfo)

	i, _ := summaryInfo[1].(map[string]interface{})
	strSpendable := i["amount_currently_spendable"].(string)
	spendable, _ := strconv.Atoi(strSpendable)
	return uint64(spendable) // unit nanogrin
}

// distribute coins when balance is > 1000 nano
func (p *payer) distribute(newBalance uint64) {
	// get a distribution table
	p.db.calcTodayRevenue(newBalance)
}

func (p *payer) watch() {
	go func() {
		ch := time.Tick(24 * time.Hour)
		for {
			select {
			case <-ch:
				newBalance := p.getNewBalance()
				if newBalance > 1000 {
					p.distribute(newBalance - 1000)
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
