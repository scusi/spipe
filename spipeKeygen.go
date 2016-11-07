package main

import (
	"crypto/rand"
	"io/ioutil"
	"log"
)

func main() {
	randbuf := make([]byte, 32)
	nBytes, err := rand.Read(randbuf)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d bytes read from random source and written to 'spipe.key'\n", nBytes)
	err = ioutil.WriteFile("spipe.key", randbuf, 0600)
	if err != nil {
		log.Fatal(err)
	}
}
