// go poc on reading from stdin and send it out to the net
package main

import(
    "os"
    "log"
    "net"
    "flag"
    "fmt"
    "bytes"
)

var remoteHost string
var remotePort string
var verbose bool

func init() {
    flag.StringVar(&remoteHost, "host", "", "remote host to connect to")
    flag.StringVar(&remotePort, "port", "", "remote port to be used")
    flag.BoolVar(&verbose, "v", true, "be verbose")
}

func checkFatal(err error) {
    if err != nil {
        log.Fatal(err)
    }
}

func check(err error) {
    if err != nil {
        log.Println(err)
    }
}

func main() {
    flag.Parse()
    endpoint := net.JoinHostPort(remoteHost, remotePort)
    conn, err := net.Dial("tcp", endpoint)
    checkFatal(err)
    if verbose == true {
        log.Printf("connected to endpoint '%s'\n", endpoint)
    }
    defer conn.Close()
    loop(conn)
}

func loop(c net.Conn) {
    for {
        // read from stdin
        inbuf := make([]byte, 1024)
        icount, err := os.Stdin.Read(inbuf)
        checkFatal(err)
        if verbose == true {
            log.Printf("read %d byte\n", icount)
            log.Printf("data read: %s\n", inbuf)
        }
        // write to network connection
        c.Write(inbuf)

        outbuf := make([]byte, 1024)
        ocount, err := c.Read(outbuf)
        checkFatal(err)
        if verbose == true {
            log.Printf("read %d byte from network\n", ocount)
        }
        if bytes.HasSuffix([]byte("\n"), outbuf) == true {
            fmt.Printf("%s", outbuf)
        }
        if bytes.HasSuffix([]byte("\n\r"), outbuf) == true {
            fmt.Printf("%s", outbuf)
        } else {
            log.Printf("%0x\n", outbuf)
            fmt.Printf("%s\n", outbuf)
        }
    }
}
