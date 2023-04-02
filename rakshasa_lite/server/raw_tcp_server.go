package server

import (
	"bytes"
	"net"
	"rakshasa_lite/common"
	"sync/atomic"
)

func (l *serverListen) Lisen() {

	for {
		c, err := l.listen.Accept()
		if err != nil {
			if err.(*net.OpError).Err == net.ErrClosed {
				return
			}

			continue
		}

		conn := &serverConnect{}
		conn.conn = c
		conn.address = c.RemoteAddr().String()
		conn.node = l.node
		conn.write = make(chan *bytes.Buffer, 64)

		if l.isSocks5 {
			conn.id = l.id
			l.node.Write(common.CMD_CONNECT_BYIDADDR_RESULT, l.replayid, append(l.randkey, l.socks5Replay...))
			go conn.handTcpReceive()
			return
		}
		conn.id = l.node.storeConn(conn)

		b := make([]byte, 4)
		b[0] = byte(conn.id)
		b[1] = byte(conn.id >> 8)
		b[2] = byte(conn.id >> 16)
		b[3] = byte(conn.id >> 24)
		conn.node.Write(common.CMD_CONNECT_BYID, l.id, append(l.randkey, b...))
		l.connMap.Store(conn.id, conn)
		go conn.handTcpReceive()

	}
}
func (l *serverListen) Close(reason string) {
	if atomic.CompareAndSwapInt32(&l.close, 0, 1) {
		if l.listen != nil {
			l.listen.Close()
		}
		l.connMap.Range(func(key, value interface{}) bool {
			if reason != remoteClose {
				l.node.Write(common.CMD_DELETE_CONNID, value.(*serverConnect).id, nil)
			}
			l.connMap.Delete(key)
			return true
		})
		if reason != remoteClose {

			l.node.Write(common.CMD_DELETE_LISTEN, l.id, l.randkey)
		}
	}
}
