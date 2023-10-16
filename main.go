package main

import (
	"log"
	"traefik-http-lxd-provider/client"
	"traefik-http-lxd-provider/worker"
)

func main() {
	w := worker.NewWorkerPool(3)
	lxd := client.NewClientConnectionPool(client.PoolConfig{})

	_, err := NewInstanceManager(w, lxd)
	if err != nil {
		log.Fatalln("error")
	}
}
