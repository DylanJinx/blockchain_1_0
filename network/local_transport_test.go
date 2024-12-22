package network

import (
	"testing"

	"github.com/stretchr/testify/assert" // go get github.com/stretchr/testify
)

func TestConnect(t *testing.T) {
	tra := NewLocalTransport("A")
	trb := NewLocalTransport("B")

	tra.Connect(trb)
	trb.Connect(tra)
	assert.Equal(t, tra.(*LocalTransport).peers[trb.Addr()], trb.(*LocalTransport))
	assert.Equal(t, trb.(*LocalTransport).peers[tra.Addr()], tra.(*LocalTransport))
}

func TestSendMessage(t *testing.T) {
	tra := NewLocalTransport("A")
	trb := NewLocalTransport("B")

	tra.Connect(trb)
	trb.Connect(tra)

	msg := []byte("hello")
	assert.Nil(t, tra.SendMessage(trb.Addr(), msg))

	rpc := <-trb.Consume()
	assert.Equal(t, rpc.From, tra.Addr())
	assert.Equal(t, rpc.Payload, msg)
}