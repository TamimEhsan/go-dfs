package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/tamimehsan/go-distributed-fs/p2p"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store  *Store
	quitCh chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:          opts.StorageRoot,
		PathTransform: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		peers:          make(map[string]p2p.Peer),
		store:          NewStore(storeOpts),
		quitCh:         make(chan struct{}),
	}
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

type MessageGetFile struct {
	Key string
}

func (s *FileServer) broadcast(msg *Message, encode bool) error {
	buf := new(bytes.Buffer)
	if encode {
		if err := gob.NewEncoder(buf).Encode(msg); err != nil {
			return err
		}
	} else {
		buf.Write(msg.Payload.([]byte))
	}

	for _, peer := range s.peers {

		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}

	}

	return nil
}

func (s *FileServer) gossip(msg *Message, peers []io.Writer, encode bool) error {
	buf := new(bytes.Buffer)
	if encode {
		if err := gob.NewEncoder(buf).Encode(msg); err != nil {
			return err
		}
	} else {
		buf.Write(msg.Payload.([]byte))
	}

	mw := io.MultiWriter(peers...)
	_, err := io.Copy(mw, buf)
	if err != nil {
		return err
	}

	return nil
}

func (s *FileServer) ReadData(key string) (int64, io.Reader, error) {

	if s.store.Exists(key) {
		fmt.Println("Whaat?")
		return s.store.Read(key)
	}

	fmt.Println("File doesn't exist on server, attempting to retrieve from network")
	p := &Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}
	err := s.broadcast(p, true)
	if err != nil {
		return 0, nil, err
	}

	time.Sleep(time.Second * 3)

	return s.store.Read(key)

}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	// read the data from the reader and
	fileBuffer := new(bytes.Buffer)
	tee := io.TeeReader(r, fileBuffer)

	// check if the file already exists
	exists := s.store.Exists(key)

	// store it in the file store
	if err := s.store.Write(key, tee); err != nil {
		return err
	}

	// if the file already exists, no need to gossip
	if exists {
		return nil
	}
	// gossip the file metadata to peers
	p := &Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: int64(fileBuffer.Len()),
		},
	}

	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	// randomly shuffle the peers
	rand.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})
	if len(peers) > 2 {
		peers = peers[:2]
	}

	err := s.gossip(p, peers, true)
	if err != nil {
		return err
	}

	// stream the file contents to peers
	time.Sleep(time.Millisecond * 5)

	mw := io.MultiWriter(peers...)
	_, err = io.Copy(mw, fileBuffer)
	if err != nil {
		return err
	}
	return nil
}

func (s *FileServer) Stop() {
	close(s.quitCh)
	fmt.Println("server signaled to stop")
}

func (s *FileServer) AddPeer(peer p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	s.peers[peer.RemoteAddr().String()] = peer
	fmt.Println("peer added: ", peer.RemoteAddr().String())
	return nil
}

func (s *FileServer) RemovePeer(peer p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	delete(s.peers, peer.RemoteAddr().String())
	fmt.Println("peer removed: ", peer.RemoteAddr().String())
	return nil
}

func (s *FileServer) bootstrapNetwork() {
	for _, peer := range s.BootstrapNodes {
		if len(peer) == 0 {
			continue
		}
		go func(peer string) {
			if err := s.Transport.Dial(peer); err != nil {
				fmt.Println("error dialing peer: ", err)
			}
		}(peer)
	}
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}
	s.bootstrapNetwork()
	s.loop()

	return nil
}

func (s *FileServer) loop() {
	defer s.Transport.Close()
	for {
		select {
		case <-s.quitCh:
			return
		case rpc := <-s.Transport.Consume():

			// decode the file metadata at first from the rpc
			var p Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&p); err != nil {
				log.Println("error decoding rpc: ", err)
			}

			s.handleMessage(rpc.From, &p)

		}
	}
}

func (s *FileServer) handleMessage(from string, msg *Message) error {
	switch msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleStoreFile(from, msg.Payload.(MessageStoreFile))
	case MessageGetFile:
		return s.handleGetFile(from, msg.Payload.(MessageGetFile))
	default:

	}
	return nil
}

func (s *FileServer) handleStoreFile(from string, msg MessageStoreFile) error {
	key := msg.Key
	fmt.Println("file metadata received on server ", s.StorageRoot, " from ", from, " : ", key)

	peer, ok := s.peers[from]
	if !ok {
		log.Println("peer not found: ", from)
		return nil
	}

	// read the file contents from the stream and write to file
	s.StoreData(key, io.LimitReader(peer, msg.Size))
	
	fmt.Println("file content received on server ", s.StorageRoot, " from ", from, " : ", key)
	// Close stream decreaments the zemaphore thus
	// the peer will end the waiting
	peer.CloseStream()
	return nil
}

func (s *FileServer) handleGetFile(from string, msg MessageGetFile) error {
	if !s.store.Exists(msg.Key) {
		fmt.Println("file doesn't exist here too!")
		return nil
	}

	peer, ok := s.peers[from]
	if !ok {
		fmt.Println("Peer not found")
		return nil
	}

	fmt.Println("file retrieved to server to network!")

	n, r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	defer r.(io.ReadCloser).Close()

	p := &Message{
		Payload: MessageStoreFile{
			Key:  msg.Key,
			Size: int64(n),
		},
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(p); err != nil {
		return err
	}

	err = peer.Send(buf.Bytes())
	if err != nil {
		return err
	}

	// stream the file contents to the peer

	_, err = io.Copy(peer, r)
	// fmt.Println("attempting to send file data:::", fileBuffer.String())
	if err != nil {
		return err
	}

	peer.CloseStream()

	return err
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
