package main

import (
	"bytes"
	"log"
	"time"

	"github.com/tamimehsan/go-distributed-fs/p2p"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandShakeFunc: p2p.NOPHandshake,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)
	FileServerOpts := FileServerOpts{
		StorageRoot:       listenAddr[1:] + "_network",
		PathTransformFunc: CASPathTransform,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(FileServerOpts)

	// tcp transport will push peers to the server
	tcpTransport.PushPeer = s.AddPeer

	return s
}

func main() {

	s1 := makeServer(":4000", "")
	s2 := makeServer(":4001", ":4000")

	go func() {
		if err := s1.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(2 * time.Second)

	go s2.Start()

	time.Sleep(2 * time.Second)

	data := bytes.NewReader([]byte("hello world"))
	s2.StoreData("hello.txt", data)

	select {}

	// go func() {
	// 	time.Sleep(5 * time.Second)
	// 	server.Stop()
	// }()

}
