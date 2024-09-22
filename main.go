package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/tamimehsan/go-distributed-fs/p2p"
)

func makeServer(listenAddr string, nodes []string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandShakeFunc: p2p.NOPHandshake,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)
	FileServerOpts := FileServerOpts{
		StorageRoot:       strings.Split(listenAddr, ":")[1] + "_network",
		PathTransformFunc: CASPathTransform,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(FileServerOpts)

	// tcp transport will push peers to the server
	tcpTransport.PushAddPeer = s.AddPeer
	tcpTransport.PushRemovePeer = s.RemovePeer

	return s
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <port>")
		os.Exit(1)
	}
	addr := os.Args[1]

	var nodes []string
	if len(os.Args) > 2 {
		nodes = os.Args[2:]
	}

	s := makeServer(addr, nodes)

	go s.Start()
	for {
		var fileName, data string
		fmt.Println("Enter file name: ")
		fmt.Scanln(&fileName)

		fmt.Println("Enter data: ")
		reader := bufio.NewReader(os.Stdin)
		data, _ = reader.ReadString('\n')

		buf := new(bytes.Buffer)
		buf.WriteString(data)
		s.StoreData(fileName, buf)
	}

}
