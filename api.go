package main

import (
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/google/logger"
	"github.com/gorilla/mux"
)

type apiServer struct {
	db   *database
	conf *config
}

func (as *apiServer) revenueHandler(w http.ResponseWriter, r *http.Request) {
	var raw []byte
	header := w.Header()
	header.Set("Content-Type", "application/json")
	header.Set("Access-Control-Allow-Origin", "*")

	table := as.db.getLastDayRevenue()
	raw, _ = json.Marshal(table)

	_, _ = w.Write(raw)
}

func (as *apiServer) sharesHandler(w http.ResponseWriter, r *http.Request) {
	var raw []byte
	header := w.Header()
	header.Set("Content-Type", "application/json")
	header.Set("Access-Control-Allow-Origin", "*")

	table := as.db.getShares()
	raw, _ = json.Marshal(table)

	_, _ = w.Write(raw)
}

func (as *apiServer) poolHandler(w http.ResponseWriter, r *http.Request) {
	var blockBatch []string
	header := w.Header()
	header.Set("Content-Type", "application/json")
	header.Set("Access-Control-Allow-Origin", "*")

	blockBatch = as.db.getAllBlockHashes()

	req, _ := http.NewRequest("GET", "http://"+as.conf.Node.Address+":"+strconv.Itoa(as.conf.Node.APIPort)+"/v1/status", nil)
	req.SetBasicAuth(as.conf.Node.AuthUser, as.conf.Node.AuthPass)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		logger.Error(err)
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Error(err)
	}
	var nodeStatus interface{}
	err = json.Unmarshal(body, &nodeStatus)
	if err != nil {
		logger.Error(err)
	}

	table := map[string]interface{}{
		"node_status":  nodeStatus,
		"mined_blocks": blockBatch,
	}

	raw, err := json.Marshal(table)
	if err != nil {
		logger.Error(err)
		return
	}

	_, _ = w.Write(raw)
}

type registerPaymentMethodForm struct {
	Pass          string `json:"pass"`
	PaymentMethod string `json:"pm"`
}

func (as *apiServer) minerHandler(w http.ResponseWriter, r *http.Request) {
	var raw []byte

	header := w.Header()
	header.Set("Content-Type", "application/json")
	header.Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	login := vars["miner_login"]

	switch r.Method {
	case "POST":
		rawBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.Error(err)
			return
		}
		var form registerPaymentMethodForm
		err = json.Unmarshal(rawBody, &form)
		if err != nil {
			logger.Error(err)
			return
		}

		if as.db.verifyMiner(login, form.Pass) == correctPassword {
			as.db.updatePayment(login, form.PaymentMethod)
			raw = []byte("{'status':'ok'}")
		} else {
			raw = []byte("{'status':'failed'}")
		}

		break
	default: // GET
		var err error
		m := as.db.getMinerStatus(login)
		raw, err = json.Marshal(m)
		if err != nil {
			logger.Error(err)
			return
		}
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
	go logger.Fatal(
		http.ListenAndServe(conf.APIServer.Address+":"+strconv.Itoa(conf.APIServer.Port), nil))
}
