package server

import (
	"cert"
	"encoding/binary"
	"errors"
	"math/rand"
	"rakshasa_lite/common"
	"time"
)

func StartRemoteSocks5(cfg *common.Addr, n *node) error {

	l := &clientListen{
		id:         common.GetID(),
		localAddr:  "",
		remoteAddr: cfg.Addr(),
		server:     n,
		typ:        "socks5",
		result:     make(chan interface{}),
		randkey:    make([]byte, 8),
	}
	binary.LittleEndian.PutUint64(l.randkey, uint64(rand.NewSource(time.Now().UnixNano()).Int63()))
	l.openOption = common.CMD_REMOTE_SOCKS5
	l.openMsg = cert.RSAEncrypterByPrivByte(append(l.randkey, cfg.String()...))
	n.Write(l.openOption, l.id, l.openMsg)
	currentNode.listenMap.Store(l.id, l)
	select {
	case res := <-l.result:
		if err, ok := res.(error); ok {
			l.Close(remoteClose)
			return err
		}
	case <-time.After(common.CMD_TIMEOUT):
		l.Close(remoteClose)
		return errors.New("time out")
	}

	return nil
}
