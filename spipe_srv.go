// spipe server 
package main

import(
    "github.com/dchest/spipe"
    "os"
    "io"
    "flag"
    "log"
    "net"
)

var remoteHost string
var remotePort string
var sharedKey string

func init() {
    flag.StringVar(&remoteHost, "host", "", "host to connect to")
    flag.StringVar(&remotePort, "port", "", "port to connect to")
    flag.StringVar(&sharedKey, "key", "", "key to be used")
}

func check(err error) {
    if err != nil {
        log.Println(err)
    }
}

func checkFatal(err error) {
    if err != nil {
        log.Fatal(err)
    }
}

func main() {
    flag.Parse()
    endpoint := net.JoinHostPort(remoteHost, remotePort)
    ln, err := spipe.Listen([]byte(sharedKey), "tcp", endpoint)
    if err != nil {
      // handle error
        checkFatal(err)
    }
    for {
        conn, err := ln.Accept()
        if err != nil {
            // handle error
            check(err)
            continue
        }
        go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    for {
        go Tunnel(conn, os.Stdout)
        go Tunnel(os.Stdin, conn)
    }
}

// Tunnel takes two io.ReadWriteCloser as argument 'from' and 'to'.
// All data recieved on the 'from' connection will be copied to the 'to' connection.
// Answers will go the exact reverse path.
// This function is used to transfer data between the incomming ssl connection and the backend forwarded to.
func Tunnel(from, to io.ReadWriteCloser) {
    written, err := io.Copy(from, to) 
    if err != nil {
        log.Printf("io.Copy Error: %s\n", err)
    }   
    log.Printf("copied %d bytes\n", written)
    from.Close()
    to.Close()
}
