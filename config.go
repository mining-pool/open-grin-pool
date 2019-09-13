package main

import (
	"encoding/json"
	"os"
)

type config struct {
	Log struct {
		Verbose   bool   `json:"verbose"`
		SystemLog bool   `json:"system_log"`
		LogFile   string `json:"log_file"`
	} `json:"log"`
	StratumServer struct {
		Address        string `json:"address"`
		Port           int    `json:"port"`
		BackupInterval string `json:"backup_interval"`
	} `json:"stratum_server"`
	APIServer struct {
		Address  string `json:"address"`
		Port     int    `json:"port"`
		AuthUser string `json:"auth_user"`
		AuthPass string `json:"auth_pass"`
	} `json:"api_server"`
	Storage struct {
		Address  string `json:"address"`
		Port     int    `json:"port"`
		Db       int    `json:"db"`
		Password string `json:"password"`
	} `json:"storage"`
	Node struct {
		Address     string `json:"address"`
		APIPort     int    `json:"api_port"`
		StratumPort int    `json:"stratum_port"`
		AuthUser    string `json:"auth_user"`
		AuthPass    string `json:"auth_pass"`
	} `json:"node"`
	Wallet struct {
		Address      string `json:"address"`
		OwnerAPIPort int    `json:"owner_api_port"`
		AuthUser     string `json:"auth_user"`
		AuthPass     string `json:"auth_pass"`
	} `json:"wallet"`
}

func parseConfig() *config {
	f, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}

	var conf config
	dec := json.NewDecoder(f)
	err = dec.Decode(&conf)
	if err != nil {
		log.Fatal(err)
	}

	return &conf
}
