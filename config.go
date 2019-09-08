package main

import (
	"encoding/json"
	"log"
	"os"
)

type config struct {
	StratumServer struct {
		Address        string `json:"address"`
		Port           int    `json:"port"`
		BackupInterval string `json:"backup_interval"`
	} `json:"stratum_server"`
	APIServer struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
	} `json:"api_server"`
	Node struct {
		Address     string `json:"address"`
		APIPort     int    `json:"api_port"`
		StratumPort int    `json:"stratum_port"`
		AuthUser    string `json:"auth_user"`
		AuthPass    string `json:"auth_pass"`
	} `json:"node"`
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
