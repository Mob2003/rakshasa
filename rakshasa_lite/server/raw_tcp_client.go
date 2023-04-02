package server

import (
	"cert"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"rakshasa_lite/common"
	"sync"
	"sync/atomic"
	"time"
)

var (
// clientListenMap = make(map[uint32]*remoteListen)
// connectMap = make(map[uint32]*rawConnect)
)

type clientListen struct {
	id         uint32
	localAddr  string
	remoteAddr string
	server     *node
	typ        string
	openOption byte
	openMsg    []byte   //掉线重连会用到
	connMap    sync.Map //clientListen关闭的时候关掉这里的id
	listen     net.Listener
	result     chan interface{}
	randkey    []byte //随机key int64
}

func StartRawBind(str string, dst []string) error {
	n, err := GetNodeFromAddrs(dst)
	if err != nil {

		return err
	}

	addrs, err := common.ResolveTCPAddr(str)
	if err != nil {

		return err
	}
	if len(addrs) != 2 {
		return errors.New("参数错误，格式为ip:port,remote_ip:remote_port")
	}

	l := &clientListen{
		id:         common.GetID(),
		localAddr:  addrs[0],
		remoteAddr: addrs[1],
		server:     n,
		typ:        "bind",
		result:     make(chan interface{}),
		openOption: common.CMD_LISTEN,
		randkey:    make([]byte, 8),
	}
	binary.LittleEndian.PutUint64(l.randkey, uint64(rand.NewSource(time.Now().UnixNano()).Int63()))
	l.openMsg = cert.RSAEncrypterByPrivByte(append(l.randkey, []byte(addrs[1])...))
	currentNode.listenMap.Store(l.id, l)
	n.Write(l.openOption, l.id, l.openMsg)
	select {
	case res := <-l.result:

		if err, ok := res.(error); ok {
			l.Close(remoteClose)
			currentNode.listenMap.Delete(l.id)
			return err
		}
	case <-time.After(common.CMD_TIMEOUT):
		l.Close(remoteClose)
		currentNode.listenMap.Delete(l.id)
		return fmt.Errorf("listen %s fail time out", addrs[1])

	}
	fmt.Println("bind 启动成功")
	//l := clientLock.Lock()

	//clientListenMap[b.id] = b
	//l.Unlock()
	return nil
}
func StartRawConnect(str string, n *node) error {
	addrs, err := common.ResolveTCPAddr(str)
	if len(addrs) != 2 || err != nil {
		return errors.New("-connect参数错误，格式为ip:port,remote_ip:remote_port")
	}

	addr1, _ := net.ResolveTCPAddr("tcp", addrs[1])
	listen, err := net.Listen("tcp", addrs[0])
	if err != nil {
		return errors.New("监听本地端口" + addrs[0] + "失败 " + err.Error())
	}

	l := &clientListen{
		id:         common.GetID(),
		localAddr:  addrs[0],
		remoteAddr: addrs[1],
		listen:     listen,
		server:     n,
		typ:        "connect",
		randkey:    make([]byte, 8),
	}
	binary.LittleEndian.PutUint64(l.randkey, uint64(rand.NewSource(time.Now().UnixNano()).Int63()))
	currentNode.listenMap.Store(l.id, l)

	go func() {
		for {
			conn, err := listen.Accept()
			if err != nil {
				if err.(*net.OpError).Err == net.ErrClosed {
					return
				}
				continue
			}

			s := &clientConnect{
				conn:    conn,
				server:  n,
				randkey: l.randkey,
			}
			s.OnOpened()
			if s.connect(common.RAW_TCP, addr1.IP.String(), uint16(addr1.Port)) {
				go rawHandleLocal(s)
			} else {
				s.Close(nodeIsClose)

			}

		}
	}()
	return nil
}
func (l *clientListen) Close(reason string) {
	l.connMap.Range(func(key, value interface{}) bool {
		value.(*clientConnect).Close(reason)
		l.connMap.Delete(key)
		return true
	})
	l.server.listenMap.Delete(l.id)
	if l.listen != nil {
		l.listen.Close()
	}
}

func rawHandleLocal(s *clientConnect) {
	buf := make([]byte, common.MAX_PLAINTEXT)

	for {
		n, err := s.conn.Read(buf[8:])
		if err != nil {

			s.Close(err.Error())
			return
		}

		var new_size int64
		if new_size = int64(common.INIT_WINDOWS_SIZE) - s.windowsSize; new_size > 0 { //扩大窗口
			atomic.AddInt64(&s.windowsSize, new_size)

		} else {
			new_size = 0
		}
		buf[0] = byte(new_size)
		buf[1] = byte(new_size >> 8)
		buf[2] = byte(new_size >> 16)
		buf[3] = byte(new_size >> 24)
		buf[4] = byte(new_size >> 32)
		buf[5] = byte(new_size >> 40)
		buf[6] = byte(new_size >> 48)
		buf[7] = byte(new_size >> 56)

		data := make([]byte, 8+n)
		copy(data, buf)
		s.server.Write(common.CMD_CONN_MSG, s.id, buf[:8+n])
	}
}
