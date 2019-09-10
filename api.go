package main

import (
	"encoding/json"
	"github.com/dgraph-io/badger"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

type apiServer struct {
	db     *badger.DB
	nc     *nodeClient
	height uint64
	conf   *config
}

func (as *apiServer) revenueHandler(w http.ResponseWriter, r *http.Request) {
	var raw []byte

	err := as.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("revenue"))
		if err != nil {
			return err
		}
		raw, err = item.ValueCopy(nil)
		return nil
	})
	if err != nil {
		log.Println(err)
	}

	header := w.Header()
	header.Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

func (as *apiServer) poolHandler(w http.ResponseWriter, r *http.Request) {
	var blockBatch []string

	err := as.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("blocks"))
		if err != nil {
			return err
		}
		raw, err := item.ValueCopy(nil)
		err = json.Unmarshal(raw, &blockBatch)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}

	req, _ := http.NewRequest("GET", "http://"+as.conf.Node.Address+":"+strconv.Itoa(as.conf.Node.APIPort)+"/v1/status", nil)
	req.SetBasicAuth(as.conf.Node.AuthUser, as.conf.Node.AuthPass)
	client := &http.Client{}
	res, _ := client.Do(req)

	dec := json.NewDecoder(res.Body)
	var nodeStatus interface{}
	_ = dec.Decode(&nodeStatus)

	table := map[string]interface{}{
		"node_status":  nodeStatus,
		"height":       as.height,
		"mined_blocks": blockBatch,
	}
	raw, err := json.Marshal(table)
	if err != nil {
		log.Println(err)
		return
	}

	header := w.Header()
	header.Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

type registerPaymentMethodForm struct {
	Pass          string `json:"pass"`
	PaymentMethod string `json:"pm"`
}

type apiResponse struct {
	Code   int    `json:"code"`
	Detail string `json:"detail"`
}

func (as *apiServer) minerHandler(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	login := vars["miner_login"]

	if r.Method == "POST" {
		raw, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			return
		}
		var form registerPaymentMethodForm
		err = json.Unmarshal(raw, &form)
		if err != nil {
			log.Println(err)
			return
		}

		var isCorrect = false
		err = as.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(login + "login"))
			if err != nil {
				if err == badger.ErrKeyNotFound {
					return nil
				}
				return err
			}

			passInDB, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			if string(passInDB) != form.Pass {
				return errWrongPassword
			} else {
				isCorrect = true
			}

			return nil
		})
		if err != nil {
			log.Println(err)
			return
		}

		if isCorrect {
			err := as.db.Update(func(txn *badger.Txn) error {
				err = txn.Set([]byte(login+"payment"), []byte(form.PaymentMethod))
				return err
			})
			if err != nil {
				login = ""
				log.Println(err)
			}
		}

		return
	}

	if r.Method == "GET" {
		err := as.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(login + "+status"))
			if err != nil {
				return err
			}
			raw, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			//raw, err := json.Marshal(res)
			//if err != nil {
			//	return err
			//}
			_, _ = w.Write(raw)
			return nil
		})
		if err != nil {
			res := map[string]string{
				"error": "no such login",
			}
			raw, err := json.Marshal(res)
			if err != nil {
				log.Println(err)
			}
			_, _ = w.Write(raw)
		}
	}

}

func (as *apiServer) loopHeight() {
	statusReq := &stratumRequest{
		ID:      "0",
		JsonRpc: "2.0",
		Method:  "height",
		Params:  nil,
	}

	ch := time.Tick(10 * time.Second)
	enc := json.NewEncoder(as.nc.c)

	for {
		select {
		case <-ch:
			err := enc.Encode(statusReq)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func initAPIServer(db *badger.DB, conf *config) {
	nc := initNodeStratumClient(conf)
	as := &apiServer{
		db:   db,
		nc:   nc,
		conf: conf,
	}

	go as.loopHeight()

	go as.nc.wait(func(sr json.RawMessage) {
		var statusRes stratumResponse
		err := json.Unmarshal(sr, &statusRes)
		if err != nil {
			log.Println(err)
		}
		if statusRes.Method == "status" {
			result, _ := statusRes.Result.(map[string]interface{})
			as.height, _ = result["height"].(uint64)
		}
	})

	r := mux.NewRouter()
	r.HandleFunc("/revenue", as.revenueHandler)
	r.HandleFunc("/pool", as.poolHandler)
	r.HandleFunc("/miner/{miner_login}", as.minerHandler)
	http.Handle("/", r)
	go log.Fatal(http.ListenAndServe(conf.APIServer.Address+":"+strconv.Itoa(conf.APIServer.Port),
		nil,
	),
	)
}
