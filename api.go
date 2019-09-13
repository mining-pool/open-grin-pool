package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type apiServer struct {
	db   *database
	conf *config
}

func (as *apiServer) revenueHandler(w http.ResponseWriter, r *http.Request) {
	var raw []byte

	table := as.db.getLastDayRevenue()
	raw, _ = json.Marshal(table)

	header := w.Header()
	header.Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

func (as *apiServer) sharesHandler(w http.ResponseWriter, r *http.Request) {
	var raw []byte

	table := as.db.getShares()
	raw, _ = json.Marshal(table)

	header := w.Header()
	header.Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

func (as *apiServer) poolHandler(w http.ResponseWriter, r *http.Request) {
	var blockBatch []string

	blockBatch = as.db.getAllBlockHashes()

	req, _ := http.NewRequest("GET", "http://"+as.conf.Node.Address+":"+strconv.Itoa(as.conf.Node.APIPort)+"/v1/status", nil)
	req.SetBasicAuth(as.conf.Node.AuthUser, as.conf.Node.AuthPass)
	client := &http.Client{}
	res, _ := client.Do(req)

	dec := json.NewDecoder(res.Body)
	var nodeStatus interface{}
	_ = dec.Decode(&nodeStatus)

	table := map[string]interface{}{
		"node_status":  nodeStatus,
		"mined_blocks": blockBatch,
	}
	raw, err := json.Marshal(table)
	if err != nil {
		log.Error(err)
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

func (as *apiServer) minerHandler(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	login := vars["miner_login"]

	var raw []byte
	switch r.Method {
	case "POST":
		raw, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error(err)
			return
		}
		var form registerPaymentMethodForm
		err = json.Unmarshal(raw, &form)
		if err != nil {
			log.Error(err)
			return
		}

		if as.db.verifyMiner(login, form.Pass) == correctPassword {
			as.db.updatePayment(login, form.PaymentMethod)
			raw = []byte("{'status':'ok'}")
		} else {
			raw = []byte("{'status':'failed'}")
		}

		return
	default:
		m := as.db.getMinerStatus(login)
		raw, _ = json.Marshal(m)
	}

	_, _ = w.Write(raw)
}

func initAPIServer(db *database, conf *config) {
	as := &apiServer{
		db:   db,
		conf: conf,
	}

	r := mux.NewRouter()
	r.HandleFunc("/pool", as.poolHandler)
	r.HandleFunc("/miner/{miner_login}", as.minerHandler)
	r.HandleFunc("/revenue", as.revenueHandler)
	r.HandleFunc("/shares", as.sharesHandler)
	http.Handle("/", r)
	go log.Fatal(http.ListenAndServe(conf.APIServer.Address+":"+strconv.Itoa(conf.APIServer.Port),
		nil,
	),
	)
}
