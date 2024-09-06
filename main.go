package main

import (
	"fmt"

	"github.com/tamimehsan/go-distributed-fs/p2p"
)

func main() {

	tcpOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":4000",
		Decoder:       p2p.DefaultDecoder{},
		HandShakeFunc: p2p.NOPHandshake,
	}

	t := p2p.NewTCPTransport(tcpOpts)
	if err := t.ListenAndAccept(); err != nil {
		fmt.Println("error: ", err)
	}
	select {}
	// run telnet localhost 4000 to check
	// if the server is listening and accepting connections

}
