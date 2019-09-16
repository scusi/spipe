// simple netcat clone in go that uses spipe to encrypt connections
package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/dchest/spipe"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
)

var mode string
var sharedKeyA string
var host string
var port string
var verbose bool

func init() {
	flag.StringVar(&mode, "m", "dial", "mode to use (listen, dial), default is: dial")
	flag.StringVar(&sharedKeyA, "k", "spipe.key", "shared key file to use")
	flag.StringVar(&host, "h", "127.0.0.1", "host to connect to")
	flag.StringVar(&port, "p", "8080", "port to connect to")
	flag.BoolVar(&verbose, "v", false, "verbose mode on if true, default: false")
	checkFlags()
}

func checkFlags() {
	flag.Parse()
	if sharedKeyA == "" {
		// generate a key and set and print it, if no key was given
		randbuf := make([]byte, 32)
		_, err := rand.Read(randbuf)
		if err != nil {
			log.Fatal(err)
		}
		sharedKeyA = fmt.Sprintf("%x", randbuf)
		log.Printf("sharedKey set to: '%s'", sharedKeyA)
	}
}

func main() {
	flag.Parse()
	// read key from file
	sharedKey, err := ioutil.ReadFile(sharedKeyA)
	if err != nil {
		log.Fatal(err)
	}
	if verbose {
		log.Printf("connection key is set to: '%x'\n", sharedKey)
	}
	hopo := net.JoinHostPort(host, port)
	switch mode {
	case "listen":
		ln, err := spipe.Listen(sharedKey, "tcp", hopo)
		if nil != err {
			log.Printf("Bind Error: %s\n", err.Error())
			return
		}
		log.Printf("listening on: %s\n", hopo)

		for {
			conn, err := ln.Accept()
			if nil != err {
				log.Printf("Accept Error: %s\n", err.Error())
				continue
			}

			tcp_con_handle(conn)
		}
	case "dial":
		conn, err := spipe.Dial(sharedKey, "tcp", hopo)
		if err != nil {
			log.Fatal(err)
		}
		tcp_con_handle(conn)
	}
}

// Handles TC connection and perform synchorinization:
// TCP -> Stdout and Stdin -> TCP
func tcp_con_handle(con net.Conn) {
	chan_to_stdout := stream_copy(con, os.Stdout)
	chan_to_remote := stream_copy(os.Stdin, con)
	select {
	case <-chan_to_stdout:
		log.Println("Remote connection is closed")
	case <-chan_to_remote:
		log.Println("Local program is terminated")
	}
}

// Performs copy operation between streams: os and tcp streams
func stream_copy(src io.Reader, dst io.Writer) <-chan int {
	buf := make([]byte, 1024)
	sync_channel := make(chan int)
	go func() {
		defer func() {
			if con, ok := dst.(net.Conn); ok {
				con.Close()
				log.Printf("Connection from %v is closed\n", con.RemoteAddr())
			}
			sync_channel <- 0 // Notify that processing is finished
		}()
		for {
			var nBytes int
			var err error
			nBytes, err = src.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Read error: %s\n", err)
				}
				break
			}
			_, err = dst.Write(buf[0:nBytes])
			if err != nil {
				log.Fatalf("Write error: %s\n", err)
			}
		}
	}()
	return sync_channel
}
