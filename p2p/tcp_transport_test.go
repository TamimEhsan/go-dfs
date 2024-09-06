package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	opts := TCPTransportOpts{
		ListenAddr:    ":4000",
		HandShakeFunc: NOPHandshake,
		Decoder:       DefaultDecoder{},
	}

	tr := NewTCPTransport(opts)
	assert.Equal(t, ":4000", tr.ListenAddr)

	assert.Nil(t, tr.ListenAndAccept())

	// t.listener.Accept()
}
