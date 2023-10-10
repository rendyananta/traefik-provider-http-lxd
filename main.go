package main

import (
	"github.com/canonical/lxd/shared/api"
	"log"
)

func main() {
	server, err := Connect()
	if err != nil {
		log.Fatalln("error")
	}

	containers, err := server.GetInstancesFull(api.InstanceTypeContainer)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("%+v", containers)
}
