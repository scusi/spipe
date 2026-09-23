// a simple spiped go implementation - compatible with the original spipe
// command from https://github.com/Tarsnap/spiped
//
package main

import (
	"flag"
	"fmt"
	"github.com/dchest/spipe"
	"github.com/google/uuid"
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
var logfile string
var verbose bool
var disableLog bool
var forwardHostPort string

// maxBytesPerSession is the maximum of bytes to be allowed to be transfered per session.
// otherwise security considerations will not hold true.
// See: https://github.com/Tarsnap/spiped/blob/master/DESIGN.md
var maxBytesPerSession = math.Pow(float64(2), float64(64))

// counter for bytes beeing transfered in this session
var transferedBytes = float64(0)

func init() {
	flag.StringVar(&mode,            "m", "listen_forward", "mode to use (listen, dial, listen_forward, dial_forward), default is: listen_forward")
	flag.StringVar(&sharedKeyA,      "k", "/etc/ssh/spiped.key", "file to read shared key from")
	flag.StringVar(&host,            "h", "", "host to connect to or listen on, default is any interface")
	flag.StringVar(&port,            "p", "8022", "port to connect to or listen on")
	flag.StringVar(&forwardHostPort, "forward", "127.0.0.1:22", "host to forward connections to")
	flag.StringVar(&logfile,         "l", "/var/log/spiped.log", "logfile to use")
	flag.BoolVar(&verbose,           "v", false, "be verbose if true, default is: false")
	flag.BoolVar(&disableLog,        "no-log", false, "disbales logging if true, default is: false")
	flag.Usage = Usage
}

const usageMSG = `
Modes:
	dial			dial to an spipe endpoint and connects stdin and stdout
	listen			start a spipe listener and connects stdin and stdout
	listen_forward	starts a spipe listener and forwards to a plaintext endpoint
	dial_forward	starts a plaintext listener and forwards to an spipe endpoint

Examples:
	// generate a key
	spipeKeygen -o spipe.key

	// start a spipe listener on 80.244.247.218:8888 and forward to 80.244.247.5:80
	spiped -m listen_forward -h 80.244.247.218 -p 8888 -forward 80.244.247.5:80 -k spipe.key

	// start a plaintext listener on 80.244.247.5:8080 and forward to spipe endpoint 80.244.247.218:8888
	spiped -m dial_forward -h 80.244.247.5 -p 8080 -forward 80.244.247.218:8888 -k spipe.key

	// recieve a file via spiped on 80.244.247.218:8080
	spiped -m listen -h 80.244.247.218 -p 8080 -k spipe.key > file

	// send a file via spiped to 80.244.247.218:8080
	cat file | spiped -m dial -h 80.244.247.218 -p 8080 -k spipe.key
`


func Usage() {
	fmt.Printf("Usage of %s: -m MODE -h HOST -p PORT -k KEY [-forward HOST:PORT]\n\nArguments:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Printf("%s\n", usageMSG)
}

func main() {
	flag.Parse()
	initLogging()
	if verbose {
		log.Printf("Maximale Anzahl übertragbarer bytes in dieser Sitzung: %.0f\n", maxBytesPerSession)
	}
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
		// listen - opens spipe listener and connects stdin and stdout to incoming connections
		ln, err := spipe.Listen(sharedKey, "tcp", hopo)
		if nil != err {
			log.Println("Bind Error!")
			return
		}

		for {
			conID := uuid.NewString()
			conn, err := ln.Accept()
			if nil != err {
				log.Println("New Session '%s'; Accept Error!\n",conID)
				continue
			}
			if verbose {
				log.Printf("New Session '%s';accepted connection from '%v'\n", conID, conn.RemoteAddr())
			}

			tcp_con_handle(conn, conID)
		}
	case "dial":
		// dial - dials to an spipe endpoint and connects stdin and stdout to connection
		connectionID := uuid.NewString()
		conn, err := spipe.Dial(sharedKey, "tcp", hopo)
		if err != nil {
			log.Fatal(err)
		}
		tcp_con_handle(conn, connectionID)
	case "listen_forward":
		// listen_forward - opens a spipe listener and forwards to plaintext tcp endpoint
		// open spipe listener
		ln, err := spipe.Listen(sharedKey, "tcp", hopo)
		if nil != err {
			log.Printf("Bind Error: %s\n", err.Error())
			return
		}

		for {
			connectionID := uuid.NewString()
			conn, err := ln.Accept()
			if nil != err {
				log.Printf("Accept Error: %s\n", err.Error())
				continue
			}
			log.Printf("Accepted Connection '%s' from %s\n", connectionID, conn.RemoteAddr())
			// dial to backend (plain tcp, no spipe)
			dst, err := net.Dial("tcp", forwardHostPort)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Forward Connection '%s' from %s to %s\n", connectionID, conn.RemoteAddr(), dst.RemoteAddr())
			tcp_con_forward(conn, dst, connectionID)
		}
	case "dial_forward":
		// dial_forward - opens a plaintext tcp listener and forwards incoming connections to a spipe endpoint
		ln, err := net.Listen("tcp", hopo)
		if nil != err {
			log.Println("Bind Error!")
			return
		}
		for {
			connectionID := uuid.NewString()
			conn, err := ln.Accept()
			if nil != err {
				log.Println("Accept Error!")
				continue
			}
			dst, err := spipe.Dial(sharedKey, "tcp", forwardHostPort)
			if err != nil {
				log.Fatal(err)
			}
			tcp_con_forward(conn, dst, connectionID)
		}
	}
}

// Handles TC connection and perform synchorinization:
// TCP -> Stdout and Stdin -> TCP
func tcp_con_handle(con net.Conn, conID string) {
	chan_to_stdout := stream_copy(con, os.Stdout, conID)
	chan_to_remote := stream_copy(os.Stdin, con, conID)
	select {
	case <-chan_to_stdout:
		log.Printf("Remote connection '%s' is closed\n", conID)
		log.Printf("Session '%s': %.0f bytes transfered\n", conID, transferedBytes)
	case <-chan_to_remote:
		log.Printf("Session '%s': Local program is terminated\n", conID)
		log.Printf("Session '%s': %.0f bytes transfered\n", conID, transferedBytes)
	}
}

// Handles TC connection and perform synchorinization:
// ---spipe---> Spiped ---TCP---> ForwardingHost
// <---spipe--- Spiped <---TCP--- ForwardingHost
func tcp_con_forward(src net.Conn, dst net.Conn, conID string) {
	chan_to_stdout := stream_copy(src, dst, conID)
	chan_to_remote := stream_copy(dst, src, conID)
	select {
	case <-chan_to_stdout:
		log.Printf("Remote connection '%s' is closed\n", conID)
	case <-chan_to_remote:
		log.Printf("Local program '%s' is terminated\n", conID)
	}
}

// Performs copy operation between streams: os and tcp streams
func stream_copy(src io.Reader, dst io.Writer, conID string) <-chan int {
	buf := make([]byte, 1024)
	sync_channel := make(chan int)
	go func() {
		defer func() {
			if con, ok := dst.(net.Conn); ok {
				con.Close()
				log.Printf("Connection '%s' from %v is closed\n", conID, con.RemoteAddr())
			}
			sync_channel <- 0 // Notify that processing is finished
		}()
		for {
			// make sure we do not transfer more than 2^64 byte per session
			if transferedBytes >= maxBytesPerSession {
				log.Printf("Session '%s': transfered bytes have reached the maximum allowed, aborting\n", conID)
				break
			}
			var nBytes int
			var err error
			nBytes, err = src.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Session '%s': Read error: %s\n", conID, err)
				} else {
					if verbose {
						log.Printf("Session '%s': %.0f bytes read\n", conID, transferedBytes)
					}
				}
				break
			}
			// count bytes transfered (read)
			transferedBytes += float64(nBytes)
			_, err = dst.Write(buf[0:nBytes])
			if err != nil {
				log.Fatalf("Session '%s': Write error: %s\n", conID, err)
			} else {
				if verbose {
					log.Printf("Session '%s': %.0f bytes written\n", conID, transferedBytes)
				}
			}
		}
	}()
	return sync_channel
}

func initLogging() {
	if !disableLog {
		log.Printf("logfile set to: '%s'\n", logfile)
		logFile, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		log.SetOutput(logFile)
	} else {
	    log.Printf("Logging is disabled!\n")
	}
}
