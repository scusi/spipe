package main

import (
	"flag"
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

func init() {
	flag.StringVar(&mode, "m", "dial", "mode to use (listen, dial), default is: dial")
	flag.StringVar(&sharedKeyA, "k", "foobarTest1234", "shared key to use")
	flag.StringVar(&host, "h", "127.0.0.1", "host to connect to")
	flag.StringVar(&port, "p", "8080", "port to connect to")
}

func main() {
	flag.Parse()
	sharedKey := []byte(sharedKeyA)
	hopo := net.JoinHostPort(host, port)
	switch mode {
	case "listen":
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
