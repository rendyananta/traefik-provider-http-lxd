package main

import (
	"context"
	lxd "github.com/canonical/lxd/client"
	"log"
	"os"
)

func Connect() (lxd.InstanceServer, error) {
	certFile, err := os.ReadFile("certs/lxd-traefik.crt")
	if err != nil {
		log.Fatal(err)
	}

	keyFile, err := os.ReadFile("certs/lxd-traefik.key")
	if err != nil {
		log.Fatal(err)
	}

	server, err := lxd.ConnectLXDWithContext(context.Background(), "https://localhost:8443", &lxd.ConnectionArgs{
		InsecureSkipVerify: true,
		TLSClientCert:      string(certFile),
		TLSClientKey:       string(keyFile),
	})

	if err != nil {
		log.Fatal(err)
	}

	return server, nil
}
