// spipeKeygen - generates a key file suitable to be used by spipe tools
// reads 32 byte from random and writes to file named 'spipe.key' in the
// local directory.
package main

import (
	"encoding/base64"
	"crypto/rand"
        "crypto/sha256"
	"io/ioutil"
	"flag"
	"fmt"
	"log"
)

var base64Out bool
var fingerprint bool
var outfile string

func init() {
	flag.BoolVar(&base64Out, "base64out", false, "if true key will be printed as base64 value and not written to file")
	flag.BoolVar(&fingerprint, "fingerprint", true, "if true fingerprint of key will be printed. Default: true")
	flag.StringVar(&outfile, "o", "spipe.key", "file output will be written to, default 'spipe.key'.")
}

func main() {
	flag.Parse()
	randbuf := make([]byte, 32)
	nBytes, err := rand.Read(randbuf)
	if err != nil {
		log.Fatal(err)
	}
	if fingerprint {
            	h := sha256.New()
		h.Write(randbuf)
		s := h.Sum(nil)
		encodedStr := base64.StdEncoding.EncodeToString(s[:32])
		fmt.Printf("Key fingerprint (base64): %s\n", encodedStr)
		fmt.Printf("Key fingerprint (hex)   : %x\n", s[:32])
        }
	if base64Out {
		encodedStr := base64.StdEncoding.EncodeToString(randbuf)
		fmt.Printf("%s\n", encodedStr)
	} else {
		log.Printf("%d bytes read from random source and written to '%s'\n", nBytes, outfile)
		err = ioutil.WriteFile(outfile, randbuf, 0600)
		if err != nil {
			log.Fatal(err)
		}
	}
}
