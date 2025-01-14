// spipeKeygen - generates a key file suitable to be used by spipe tools
// reads 32 byte from random and writes to file named 'spipe.key' in the
// local directory.
package main

import (
	"crypto/rand"
	"io/ioutil"
	"flag"
	"fmt"
	"log"
)

var base64Out bool
var outfile string

func init() {
	flag.BoolVar(&base64Out, "base64out", false, "if true key will be printed as base64 value and not written to file")
	flag.StringVar(&outfile, "o", "spipe.key", "file output will be written to, default 'spipe.key'.")
}

func main() {
	flag.Parse()
	randbuf := make([]byte, 32)
	nBytes, err := rand.Read(randbuf)
	if err != nil {
		log.Fatal(err)
	}
	if base64Out {
		fmt.Printf("key: %x\n", randbuf)
	} else {
		log.Printf("%d bytes read from random source and written to 'spipe.key'\n", nBytes)
		err = ioutil.WriteFile(outfile, randbuf, 0600)
		if err != nil {
			log.Fatal(err)
		}
	}
}
