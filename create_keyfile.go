package main

import(
    "io/ioutil"
    "crypto/rand"
    "log"
)

func main() {
   randbuf := make([]byte, 32) 
   nBytes, err := rand.Read(randbuf)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("%d bytes read from random source\n", nBytes)
    err = ioutil.WriteFile("keyfile", randbuf, 0600)
    if err != nil {
        log.Fatal(err)
    }
}
