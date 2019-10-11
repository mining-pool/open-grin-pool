package main

import (
	"io/ioutil"
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

func (p *payer) getNewBalance() uint64 {
	req, _ := http.NewRequest("GET", "http://"+p.conf.Wallet.Address+":"+strconv.Itoa(p.conf.Wallet.OwnerAPIPort)+"/v1/wallet/owner/retrieve_summary_info?refresh", nil)
	req.SetBasicAuth(p.conf.Node.AuthUser, p.conf.Node.AuthPass)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		logger.Error("failed to get balance from wallet, treat this as no income")
		return 0
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Error(err)
	}
	var summaryInfo []interface{}
	_ = json.Unmarshal(body, &summaryInfo)

	i, ok := summaryInfo[1].(map[string]interface{})
	if !ok {
		logger.Error("failed to format summaryInfo")
	}
	strSpendable, ok := i["amount_currently_spendable"].(string)
	if !ok {
		logger.Error("failed to format amount_currently_spendable")
	}
	spendable, err := strconv.ParseUint(strSpendable, 10, 64)
	if err != nil {
		logger.Error(err)
	}

	return spendable // unit nanogrin
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
				newBalance := p.getNewBalance()
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
