package main

import (
	"context"
	lxd "github.com/canonical/lxd/client"
	"log"
	"os"
)

func main() {
	certFile, err := os.ReadFile("certs/lxd-traefik.crt")

	if err != nil {
		log.Fatal(err)
	}

	keyFile, err := os.ReadFile("certs/lxd-traefik.key")

	if err != nil {
		log.Fatal(err)
	}

	server, err := lxd.ConnectPublicLXDWithContext(context.Background(), "https://192.168.64.13:8443", &lxd.ConnectionArgs{
		TLSClientCert: string(certFile),
		TLSClientKey:  string(keyFile),
	})

	if err != nil {
		log.Fatal(err)
	}

	log.Println(server)
}
