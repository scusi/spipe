package main

import (
	"flag"
	"fmt"
	"github.com/dchest/spipe"
	"io"
	"log"
	"net"
	"os"
)

var mode string
var sharedKeyA string
var host string
var port string
var forwardHostPort string

func init() {
	flag.StringVar(&mode, "m", "listen", "mode to use (listen, dial, listen_forward, dial_forward), default is: listen")
	flag.StringVar(&sharedKeyA, "k", "foobarTest1234", "shared key to use")
	flag.StringVar(&host, "h", "127.0.0.1", "host to connect to or listen on")
	flag.StringVar(&port, "p", "8080", "port to connect to or listen on")
	flag.StringVar(&forwardHostPort, "forward", "127.0.0.1:22", "host to forward connections to")
	flag.Usage = Usage
}

const usageMSG = `
Modes:
	dial			dial to an spipe endpoint and connects stdin and stdout
	listen			start a spipe listener and connects stdin and stdout
	listen_forward	starts a spipe listener and forwards to a plaintext endpoint
	dial_forward	starts a plaintext listener and forwards to an spipe endpoint

Examples:

	// start a spipe listener on 80.244.247.218:8888 and forward to 80.244.247.5:80
	spiped -m listen_forward -h 80.244.247.218 -p 8888 -forward 80.244.247.5:80 -k 9jfdf807n987976xnwfru897234ÖUJDEUW

	// start a plaintext listener on 80.244.247.5:8080 and forward to spipe endpoint 80.244.247.218:8888
	spiped -m dial_forward -h 80.244.247.5 -p 8080 -forward 80.244.247.218:8888 -k testtestetst

	// recieve a file via spiped on 80.244.247.218:8080
	spiped -m listen -h 80.244.247.218 -p 8080 -k testtesttest > file

	// send a file via spiped to 80.244.247.218:8080
	cat file | spiped -m dial -h 80.244.247.218 -p 8080 -k testtesttest
`

func Usage() {
	fmt.Printf("Usage of %s: -m MODE -h HOST -p PORT -k KEY [-forward HOST:PORT]\n\nArguments:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Printf("%s\n", usageMSG)
}

func main() {
	flag.Parse()
	sharedKey := []byte(sharedKeyA)
	hopo := net.JoinHostPort(host, port)
	switch mode {
	case "listen":
		// listen - opens spipe listener and connects stdin and stdout to incoming connections
		ln, err := spipe.Listen(sharedKey, "tcp", hopo)
		if nil != err {
			log.Println("Bind Error!")
			return
		}

		for {
			conn, err := ln.Accept()
			if nil != err {
				log.Println("Accept Error!")
				continue
			}

			tcp_con_handle(conn)
		}
	case "dial":
		// dial - dials to an spipe endpoint and connects stdin and stdout to connection
		conn, err := spipe.Dial(sharedKey, "tcp", hopo)
		if err != nil {
			log.Fatal(err)
		}
		tcp_con_handle(conn)
	case "listen_forward":
		// listen_forward - opens a spipe listener and forwards to plaintext tcp endpoint
		// open spipe listener
		ln, err := spipe.Listen(sharedKey, "tcp", hopo)
		if nil != err {
			log.Println("Bind Error!")
			return
		}

		for {
			conn, err := ln.Accept()
			if nil != err {
				log.Println("Accept Error!")
				continue
			}
			// dial to backend (plain tcp, no spipe)
			dst, err := net.Dial("tcp", forwardHostPort)
			if err != nil {
				log.Fatal(err)
			}

			tcp_con_forward(conn, dst)
		}
	case "dial_forward":
		// dial_forward - opens a plaintext tcp listener and forwards incoming connections to a spipe endpoint
		ln, err := net.Listen("tcp", hopo)
		if nil != err {
			log.Println("Bind Error!")
			return
		}
		for {
			conn, err := ln.Accept()
			if nil != err {
				log.Println("Accept Error!")
				continue
			}
			dst, err := spipe.Dial(sharedKey, "tcp", forwardHostPort)
			if err != nil {
				log.Fatal(err)
			}
			tcp_con_forward(conn, dst)
		}
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

// Handles TC connection and perform synchorinization:
// ---spipe---> Spiped ---TCP---> ForwardingHost
// <---spipe--- Spiped <---TCP--- ForwardingHost
func tcp_con_forward(src net.Conn, dst net.Conn) {
	chan_to_stdout := stream_copy(src, dst)
	chan_to_remote := stream_copy(dst, src)
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
