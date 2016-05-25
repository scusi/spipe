// a simple spiped go implementation - compatible with the original spipe command from https://github.com/Tarsnap/spiped
//
package main

import (
	"flag"
	"github.com/dchest/spipe"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"os"
)

var mode string
var sharedKeyA string
var host string
var port string
var verbose bool

// maxBytesPerSession is the maximum of bytes to be allowed to be transfered per session.
// otherwise security considerations will not hold true.
// See: https://github.com/Tarsnap/spiped/blob/master/DESIGN.md
var maxBytesPerSession = math.Pow(float64(2), float64(64))

// counter for bytes beeing transfered in this session
var transferedBytes = float64(0)

func init() {
	flag.StringVar(&mode, "m", "listen", "mode to use (listen, dial), default is: listen")
	flag.StringVar(&sharedKeyA, "k", "keyfile", "shared key to use")
	flag.StringVar(&host, "h", "127.0.0.1", "host to connect to")
	flag.StringVar(&port, "p", "8080", "port to connect to")
	flag.BoolVar(&verbose, "v", false, "be verbose if true, default is: false")
}

func main() {
	flag.Parse()
	if verbose {
		log.Printf("Maximale Anzahl übertragbarer bytes in dieser Sitzung: %.0f\n", maxBytesPerSession)
	}
	// read key from file
	sharedKey, err := ioutil.ReadFile(sharedKeyA)
	if err != nil {
		log.Fatal(err)
	}
	// read key from string
	//sharedKey := []byte(sharedKeyA)
	if verbose {
		log.Printf("connection key is set to: '%x'\n", sharedKey)
	}
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
			if verbose {
				log.Printf("accepted connection from '%v'\n", conn.RemoteAddr())
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
		log.Printf("%.0f bytes transfered\n", transferedBytes)
	case <-chan_to_remote:
		log.Println("Local program is terminated")
		log.Printf("%.0f bytes transfered\n", transferedBytes)
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
			// make sure we do not transfer more than 2^64 byte per session
			if transferedBytes >= maxBytesPerSession {
				log.Println("transfered bytes have reached the maximum allowed, aborting")
				break
			}
			var nBytes int
			var err error
			nBytes, err = src.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Read error: %s\n", err)
				}
				break
			}
			// count bytes transfered (read)
			transferedBytes += float64(nBytes)
			_, err = dst.Write(buf[0:nBytes])
			if err != nil {
				log.Fatalf("Write error: %s\n", err)
			}
		}
	}()
	return sync_channel
}
