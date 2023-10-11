package main

import (
	"log"
)

func main() {
	_, err := NewInstanceManager()
	if err != nil {
		log.Fatalln("error")
	}
}
