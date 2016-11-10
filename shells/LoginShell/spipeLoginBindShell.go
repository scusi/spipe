// a simple login shell via spipes
//
// known limitations:
//  - echo is not set to off for password input
//  - you have a shell after authentication, but no promt.
//  - PAM authentication is only working for the user this prog runs as
//    Unless run as priviledged user such as root, WHICH IS NOT RECOMMENDED!
//
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"github.com/dchest/spipe"
	"github.com/msteinert/pam"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
)

var address string // listening address, e.g. 127.0.0.1:8080
var keyFile string // file to read key from, use spipeKeygen.go to generate a key file

func init() {
	flag.StringVar(&address, "l", ":7777", "adress to bind to")
	flag.StringVar(&keyFile, "k", "spipe.key", "file to read key from")
}

func LoginShell(sharedKey []byte, network, address string) {
	l, _ := spipe.Listen(sharedKey, network, address)
	defer l.Close()
	for {
		// Wait for a connection.
		conn, _ := l.Accept()
		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go func(c net.Conn) {
			t, err := pam.StartFunc("", "", func(s pam.Style, msg string) (string, error) {
				switch s {
				case pam.PromptEchoOff:
					fmt.Fprint(c, msg+" ")
					input, err := bufio.NewReader(c).ReadString('\n')
					if err != nil {
						return "", err
					}
					return input[:len(input)-1], nil
				case pam.PromptEchoOn:
					fmt.Fprint(c, msg+" ")
					input, err := bufio.NewReader(c).ReadString('\n')
					if err != nil {
						return "", err
					}
					return input[:len(input)-1], nil
				case pam.ErrorMsg:
					log.Print(c, msg)
					return "", nil
				case pam.TextInfo:
					fmt.Fprintln(c, msg)
					return "", nil
				}
				return "", errors.New("Unrecognized message style")
			})
			if err != nil {
				log.Fatalf("Start: %s", err.Error())
			}
			err = t.Authenticate(0)
			if err != nil {
				log.Fatalf("Authenticate: %s", err.Error())
			}
			fmt.Fprintln(c, "Authentication succeeded!")
			fmt.Fprintln(c, "starting a shell...")
			cmd := exec.Command("/bin/bash", "--login")
			cmd.Stdin = c
			cmd.Stdout = c
			cmd.Stderr = c
			cmd.Run()
			defer c.Close()
		}(conn)
	}
}

func main() {
	flag.Parse()
	sharedKey, err := ioutil.ReadFile(keyFile)
	if err != nil {
		panic(err)
	}
	LoginShell(sharedKey, "tcp", address)
}
