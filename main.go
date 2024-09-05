package main

import (
	"fmt"

	"github.com/tamimehsan/go-distributed-fs/p2p"
)

func main() {

	t := p2p.NewTCPTransport(":4000")
	if err := t.ListenAndAccept(); err != nil {
		fmt.Println("error: ", err)
	}
	select {}
	// run telnet localhost 4000 to check
	// if the server is listening and accepting connections

}
