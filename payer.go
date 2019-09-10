package main

import (
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type payer struct {
	db   *badger.DB
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
func (p *payer) distribute(newBalance uint64) map[string]uint64 {
	// get a distribution table
	shareTable := make(map[string]uint64)
	var totalShare uint64
	toSendTable := make(map[string]uint64)
	_ = p.db.Update(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte("shares+")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			login := strings.Trim(string(item.Key()), string(prefix))
			_ = item.Value(func(v []byte) error {
				shareTable[login] = new(big.Int).SetBytes(v).Uint64()
				totalShare = totalShare + shareTable[login]
				return nil
			})

			_ = txn.Set(item.Key(), big.NewInt(0).Bytes())

		}

		for login, shares := range shareTable {
			toSendTable[login] = shares / totalShare * newBalance
		}

		api := map[string]interface{}{
			"date":  time.Now().Month().String() + time.Now().Weekday().String(),
			"sheet": toSendTable,
		}

		raw, _ := json.Marshal(api)

		_ = txn.Set([]byte("revenue"), raw)

		return nil
	})

	return toSendTable
}

func (p *payer) watch() {
	go func() {
		ch := time.Tick(24 * time.Hour)
		for {
			select {
			case <-ch:
				newBalance := p.getNewBalance()
				if newBalance > 1000 {
					table := p.distribute(newBalance - 1000)
					save(table)
				}
			}
		}

	}()
}

func save(m map[string]uint64) {
	newFileName := "PAYMENT" + "-" + strconv.Itoa(time.Now().Year()) + "-" +
		time.Now().Month().String() + "-" + strconv.Itoa(time.Now().Day())
	f, err := os.Create(newFileName + ".csv")
	defer f.Close()
	if err != nil {
		log.Println(err)
	}

	for k, v := range m {
		_, err = fmt.Fprintf(f, "%s %d\n", k, v)
	}
}
