// spipe client
package main

import(
    "github.com/dchest/spipe"
    //"bytes"
    "flag"
    "log"
    "net"
    "fmt"
)

var remoteHost string
var remotePort string
var sharedKey string

func init() {
    flag.StringVar(&remoteHost, "host", "", "host to connect to")
    flag.StringVar(&remotePort, "port", "", "port to connect to")
    flag.StringVar(&sharedKey, "key", "", "key to use")
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
    conn, err := spipe.Dial([]byte(sharedKey), "tcp", endpoint)
    if err != nil {
        //handle erroor
        checkFatal(err)
    }
    fmt.Fprintf(conn, "Hello wie geht es denn so da drüben.\n") 
    //var buf bytes.Buffer
    buf := make([]byte, 1024)
    i, err := conn.Read(buf)
    if err != nil {
        check(err)
    }
    log.Printf("read bytes: %d\n", i)
    log.Printf("read data: %s\n", buf)
}
