package main

func main() {
	conf := parseConfig()
	db := initDB()
	go initAPIServer(db, conf)
	go initStratumServer(db, conf)
	for {
		select {}
	}
}
