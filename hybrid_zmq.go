package hybridpipe

import (
	"github.com/pebbe/zmq4"
)

type ZMQPacket struct {
	dConn *zmq4.Socket
	aConn *zmq4.Socket
}

func (zp *ZMQPacket) Connect() error {

	return nil
}

func (zp *ZMQPacket) Dispatch(pipe string, d interface{}) error {

	return nil
}

func (zp *ZMQPacket) Accept(pipe string, fn Process) error {

	return nil
}

func (zp *ZMQPacket) Remove(pipe string) error {
	return nil
}

func (zp *ZMQPacket) Close() {

}
