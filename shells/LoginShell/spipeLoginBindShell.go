//package shells
package main

import (
	"bufio"
	"errors"
	"fmt"
	//"github.com/bgentry/speakeasy"
	"github.com/dchest/spipe"
	"github.com/msteinert/pam"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
)

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
					//return speakeasy.FAsk(c, msg)
					//return speakeasy.Ask(msg)
					/*
					 */
					fmt.Fprint(c, msg+" ")
					input, err := bufio.NewReader(c).ReadString('\n')
					if err != nil {
						return "", err
					}
					//log.Printf("PromptEchoOff read: %s", input)
					return input[:len(input)-1], nil
				case pam.PromptEchoOn:
					fmt.Fprint(c, msg+" ")
					input, err := bufio.NewReader(c).ReadString('\n')
					if err != nil {
						return "", err
					}
					//log.Printf("PromptEchoOn read: %s\n", input)
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
	sharedKey, err := ioutil.ReadFile("spipe.key")
	if err != nil {
		panic(err)
	}
	LoginShell(sharedKey, "tcp", ":8000")
}
