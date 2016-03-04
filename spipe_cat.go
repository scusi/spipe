// netcat on spipes
package main

import(
    "io"
    "github.com/dchest/spipe"
    //"bytes"
    "flag"
    "log"
    "net"
    "os"
)

var remoteHost string
var remotePort string
var sharedKey string

func init() {
    flag.StringVar(&remoteHost, "host", "", "host to connect to")
    flag.StringVar(&remotePort, "port", "", "port to connect to")
    flag.StringVar(&sharedKey,  "key", "", "key to use")
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
    // open an reader on stdin
    endpoint := net.JoinHostPort(remoteHost, remotePort)
    conn, err := spipe.Dial([]byte(sharedKey), "tcp", endpoint)
    if err != nil {
        //handle erroor
        checkFatal(err)
    }
    for {
	    go Tunnel(os.Stdin, conn)
	    go Tunnel(conn, os.Stdout)
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

