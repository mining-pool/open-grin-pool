package main

import (
	"errors"
	"github.com/dgraph-io/badger"
	"log"
)

var errWrongPassword = errors.New("wrong password")

func initDB() *badger.DB {
	db, err := badger.Open(badger.DefaultOptions("./poolDB"))
	if err != nil {
		log.Fatal(err)
	}

	return db
}
