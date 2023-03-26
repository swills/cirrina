package main

import (
	"log"
	"time"
)

const (
	port = ":50051"
)

func main() {
	log.Print("Starting daemon")
	go rpcServer()
	go processRequests()
	for {
		time.Sleep(1 * time.Second)
	}
}
