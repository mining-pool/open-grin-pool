package main

import (
	"strconv"
	"strings"
	"time"
)

type payer struct {
	db    *database
	conf  *config
	owner *OwnerAPI
}

type jsonRPCResponse struct {
	ID      string                 `json:"id"`
	JsonRpc string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Result  interface{}            `json:"result"`
	Error   map[string]interface{} `json:"error"`
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
			log.Error(err)
		}
		min, err := strconv.Atoi(m[1])
		if err != nil {
			log.Error(err)
		}

		var getNewBalance func() uint64
		switch p.conf.Wallet.OwnerAPIVersion {
		case "v1":
			getNewBalance = p.owner.getNewBalanceV1
		case "v2":
			getNewBalance = p.owner.getNewBalanceV2
		case "v3":
			getNewBalance = p.owner.getNewBalanceV3
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
		db:    db,
		conf:  conf,
		owner: NewOwnerAPI(db, conf),
	}
	p.watch()

	return p
}
