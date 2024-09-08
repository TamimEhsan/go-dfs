package main

import (
	"log"
	"time"

	"github.com/tamimehsan/go-distributed-fs/p2p"
)

func main() {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandShakeFunc: p2p.NOPHandshake,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)
	FileServerOpts := FileServerOpts{
		StorageRoot:       "3000_files",
		PathTransformFunc: CASPathTransform,
		Transport:         tcpTransport,
	}

	server := NewFileServer(FileServerOpts)
	go func() {
		time.Sleep(5 * time.Second)
		server.Stop()
	}()
	if err := server.Start(); err != nil {
		log.Fatalf("error starting server: %v", err)
	}

}
