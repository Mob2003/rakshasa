package server

import (
	"bytes"
	"cert"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"rakshasa_lite/common"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	UDP_PORT_MIN    = 30000
	UDP_PORT_MAX    = 60000
	SOCKES5_VERSION = 5
)

var (
	SOCKES5_AUTH_SUSSCES   []byte = []byte{5, 0}
	SOCKES5_AUTH_SUSSCES_PASSWD []byte = []byte{5, 2}
	PROTOCOL_ERR = errors.New("protocolErr")
)

const (
	CONN_AUTH_CLOSE   = 0
	CONN_AUTH_NONE    = 1
	CONN_AUTH_PW      = 2
	CONN_AUTH_OK      = 3
	CONN_AUTH_MESSAGE = 4
	CONN_REMOTE_CLOSE = 0
	CONN_REMOTE_OPEN  = 1
)

type clientConnect struct {
	cfg         *common.Addr
	windowsSize int64
	isClose     int32
	conn        net.Conn
	udpConn     net.Conn

	remote int32
	auth   int
	server *node
	id     uint32
	wait   chan int
	close  string

	udpMap     sync.Map
	udpRepData []byte
	addrData   []byte

	listenId uint32
	randkey  []byte
}

func (s *clientConnect) Write(b []byte) {

	switch b[0] {

	case common.CMD_CONNECT_BYIDADDR_RESULT:

		switch common.NetWork(b[9]) {
		case common.SOCKS5_CMD_CONNECT:

			if b[10] != 1 {
				go func() { s.Close("") }()
			} else {

				//发送成功消息
				s.auth = CONN_AUTH_MESSAGE
				s.conn.Write(append([]byte{5, 0, 0}, s.addrData...))
			}
		case common.SOCKS5_CMD_BIND:
			s.auth = CONN_AUTH_MESSAGE
			s.conn.Write(append([]byte{5, 0, 0}, s.addrData...))
		case common.RAW_TCP:
			if b[10] != 1 {
				go func() { s.Close("") }()
			}
		default:
			log.Println("socks5 未处理")
		}

	case common.CMD_CONN_MSG:

		s.conn.Write(b[1:])
		s.Addwindow(int64(-len(b[1:])))
	case common.CMD_CONN_UDP_MSG:
		s.udpConn.Write(b[1:])
	}

}

var remoteClose = "服务器要求远程关闭"
var nodeIsClose = "节点已经断开连接"

func (s *clientConnect) Close(msg string) {
	if atomic.CompareAndSwapInt32(&s.isClose, 0, 1) {

		<-s.wait
		s.wait <- common.CONN_STATUS_CLOSE
		s.auth = CONN_AUTH_CLOSE
		s.server.connMap.Delete(s.id)

		if msg == "" {
			msg = "未知关闭"
		}
		s.close = msg
		if msg == remoteClose {
			s.remote = CONN_REMOTE_CLOSE
		} else if s.remote == CONN_REMOTE_OPEN {
			s.remote = CONN_REMOTE_CLOSE
			s.Remoteclose()
		}

		s.conn.Close()
		if s.udpConn != nil {
			s.udpConn.Close()
		}
		s.udpMap.Range(func(k, _ interface{}) bool {
			s.udpMap.Delete(k)
			return true
		})
	}

}
func (s *clientConnect) Addwindow(window int64) {

	windows_size := atomic.AddInt64(&s.windowsSize, window)
	windows_update_size := int64(common.INIT_WINDOWS_SIZE)

	if windows_size < windows_update_size/2 { //扩大窗口
		if size := windows_update_size - s.windowsSize; size > 0 {
			atomic.AddInt64(&s.windowsSize, size)

			go func() {
				buf := make([]byte, 8)
				buf[0] = byte(size & 255)
				buf[1] = byte(size >> 8 & 255)
				buf[2] = byte(size >> 16 & 255)
				buf[3] = byte(size >> 24 & 255)
				buf[4] = byte(size >> 32 & 255)
				buf[5] = byte(size >> 40 & 255)
				buf[6] = byte(size >> 48 & 255)
				buf[7] = byte(size >> 56 & 255)
				s.server.Write(common.CMD_WINDOWS_UPDATE, s.id, buf)
			}()
		}
	}
}

func StartSocks5(cfg *common.Addr, dst []string) error {
	var target *node
	var err error
	if len(dst) == 0 {
		target = currentNode
	} else {
		target, err = GetNodeFromAddrs(dst)
		if err != nil {
			return err
		}
	}

	l := &clientListen{

		server:    target,
		localAddr: cfg.Addr(),
		id:        common.GetID(),
		typ:       "socks5",
		randkey:   make([]byte, 8),
	}
	binary.LittleEndian.PutUint64(l.randkey, uint64(rand.NewSource(time.Now().UnixNano()).Int63()))
	l.listen, err = StartSocks5WithServer(cfg, target, l.id)
	if err != nil {

		return err
	}

	currentNode.listenMap.Store(l.id, l)
	return nil
}
func StartSocks5WithServer(cfg *common.Addr, n *node, id uint32) (net.Listener, error) {
	l, err := net.Listen("tcp", cfg.Addr())
	if err != nil {
		return nil, err
	}
	randkey := make([]byte, 8)
	binary.LittleEndian.PutUint64(randkey, uint64(rand.NewSource(time.Now().UnixNano()).Int63()))
	fmt.Println("socks5 start ", cfg.Addr())
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				if err.(*net.OpError).Err == net.ErrClosed {
					return
				}
				continue
			}

			c := &clientConnect{
				cfg:      cfg,
				conn:     conn,
				server:   n,
				listenId: id,
				randkey:  randkey,
			}

			go handleSocks5Local(c)

		}
	}()
	return l, nil
}

func (s *clientConnect) OnOpened() (close bool) {
	s.wait = make(chan int, 1)
	s.auth = CONN_AUTH_NONE
	s.remote = CONN_REMOTE_OPEN
	s.windowsSize = 0
	s.wait <- common.CONN_STATUS_OK

	return
}

// 监听本地服务
func handleSocks5Local(s *clientConnect) {
	defer func() {
		if err := recover(); err != nil {

		}
	}()
	b := make([]byte, common.MAX_PLAINTEXT-8)
	if s.OnOpened() {
		s.Close("无法获得服务器连接")
	}
	for {
		n, err := s.conn.Read(b)
		if err != nil {

			s.Close(err.Error())
			return
		}
		data := b[:n]

		switch s.auth {
		case CONN_AUTH_NONE:

			if len(data) > 2 {
				if data[0] == 5 {
					if s.cfg.User() != "" && s.cfg.Password() != "" {
						s.conn.Write(SOCKES5_AUTH_SUSSCES_PASSWD)
						s.auth = CONN_AUTH_PW
					} else {
						s.conn.Write(SOCKES5_AUTH_SUSSCES)
						s.auth = CONN_AUTH_OK
					}

				}

			}
		case CONN_AUTH_PW:

			if s.cfg.User() != "" && s.cfg.Password() != "" {
				if len(data) > 4 {
					defer recover()
					user := string(data[2 : 2+data[1]])
					password := string(data[3+data[1] : 3+data[1]+data[2+data[1]]])
					if user == s.cfg.User() && password == s.cfg.Password() {
						s.conn.Write([]byte{5, 0})

						s.auth = CONN_AUTH_OK
					} else {
						s.conn.Write([]byte{5, 1})
					}
				}
			} else {
				s.conn.Write([]byte{5, 0})
				s.auth = CONN_AUTH_OK
			}

		case CONN_AUTH_OK:

			s.addrData = data[3:]
			switch common.NetWork(data[1]) {
			case common.SOCKS5_CMD_CONNECT:
				addr, port := socks5ReadAddr(data)
				if !s.connect(common.SOCKS5_CMD_CONNECT, addr, port) {
					s.Close(nodeIsClose)
				}
			case common.SOCKS5_CMD_BIND:
				addr, port := socks5ReadAddr(data)
				if !s.connect(common.SOCKS5_CMD_BIND, addr, port) {
					s.Close(nodeIsClose)
				}
			case common.SOCKS5_CMD_UDP:

				localIP := s.conn.LocalAddr().String()
				localIP = localIP[:strings.Index(localIP, ":")]
				//找一个能用的udp端口
				var port uint16
				for i := uint16(UDP_PORT_MIN); i <= UDP_PORT_MAX; i++ {
					s.udpConn, err = net.ListenUDP("udp", &net.UDPAddr{
						IP:   net.ParseIP(localIP),
						Port: int(i),
					})
					if err == nil {
						port = i
						break
					}
				}
				if s.udpConn == nil {
					data[0] = 5
					data[1] = 1 //RepRuleFailure
					s.conn.Write(data)
					continue
				}
				repdata := []byte{5, 0, 0, 1, 0, 0, 0, 0, byte(port >> 8), byte(port)}
				ipb := ipToByte(localIP)
				addr, port := socks5ReadAddr(data)

				if s.connect(common.SOCKS5_CMD_UDP, addr, port) {
					copy(repdata[4:], ipb)
					s.conn.Write(repdata)
					go handleSocks5Udp(s)
				} else {
					s.Close(nodeIsClose)
				}
			default:
				data[0] = 5
				data[1] = 7 //RepCmdNotSupported
				s.conn.Write(data)
			}

		case CONN_AUTH_MESSAGE:

			//binary.LittleEndian.PutUint32(outbuf[5:], crc32.ChecksumIEEE(data)+conn.msgno)
			//conn.msgno++

			var new_size int64
			if new_size = int64(common.INIT_WINDOWS_SIZE) - s.windowsSize; new_size > 0 { //扩大窗口
				atomic.AddInt64(&s.windowsSize, new_size)

			} else {
				new_size = 0
			}
			buf := make([]byte, 8)
			buf[0] = byte(new_size)
			buf[1] = byte(new_size >> 8)
			buf[2] = byte(new_size >> 16)
			buf[3] = byte(new_size >> 24)
			buf[4] = byte(new_size >> 32)
			buf[5] = byte(new_size >> 40)
			buf[6] = byte(new_size >> 48)
			buf[7] = byte(new_size >> 56)

			s.server.Write(common.CMD_CONN_MSG, s.id, append(buf, data...))
		}
	}

}
func handleSocks5Udp(s *clientConnect) {
	var b = make([]byte, 65535)
	for {
		n, err := s.udpConn.Read(b)
		if err != nil {
			s.Close(err.Error())
			return
		}
		data := b[:n]
		if b[2] != 0 {
			//不支持分片
			continue
		}

		data = data[3:]
		common.GetIDLock.Lock()
		var udpid uint32
		switch data[0] {
		case 1:
			ip := fmt.Sprintf("%d.%d.%d.%d:%d", data[1], data[2], data[3], data[4], int(data[5])<<8|int(data[6]))
			if v, ok := s.udpMap.Load(ip); !ok {

				udps := &clientConnect{
					server:  s.server,
					randkey: s.randkey,
				}
				udps.udpConn = s.udpConn
				udps.id = udps.server.storeConn(s)
				udpid = udps.id
				udps.udpRepData = make([]byte, 10)
				copy(udps.udpRepData, data)
				udps.udpMap.Store(ip, udpid)
			} else {
				udpid = v.(uint32)
			}
		case 3:
		case 4:
		}
		common.GetIDLock.Unlock()
		buf := make([]byte, 4)
		buf[0] = byte(udpid)
		buf[1] = byte(udpid >> 8)
		buf[2] = byte(udpid >> 16)
		buf[3] = byte(udpid >> 24)
		s.server.Write(common.CMD_CONN_UDP_MSG, udpid, append(buf, data...))
	}

}
func (s *clientConnect) connect(command common.NetWork, addr string, port uint16) bool {
	if atomic.LoadInt32(&s.server.isClose) == 1 {
		s.server, _ = GetNodeFromAddrs(s.server.reConnectAddrs)
	}
	if atomic.LoadInt32(&s.server.isClose) == 1 {
		return false
	}
	ports := strconv.Itoa(int(port))
	buf := make([]byte, 2+len(addr)+len(ports))
	s.id = s.server.storeConn(s)
	buf[0] = byte(command)
	copy(buf[1:], addr)
	buf[1+len(addr)] = ':'
	copy(buf[2+len(addr):], ports)
	s.server.Write(common.CMD_CONNECT_BYIDADDR, s.id, cert.RSAEncrypterByPrivByte(append(s.randkey, buf...)))
	if value, ok := s.server.listenMap.Load(s.listenId); ok {
		switch v := value.(type) {
		case *serverListen:
			v.connMap.Store(s.id, s)
		case *clientListen:
			v.connMap.Store(s.id, s)
		}
	}
	return true
}

func Bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func (s *clientConnect) Remoteclose() {

	s.close = "本地要求远程关闭"

	buf := make([]byte, 4)
	buf[0] = byte(s.id)
	buf[1] = byte(s.id >> 8)
	buf[2] = byte(s.id >> 16)
	buf[3] = byte(s.id >> 24)
	s.server.Write(common.CMD_DELETE_LISTENCONN_BYID, s.listenId, append(s.randkey, buf...))

}

func ipToByte(ip string) []byte {
	var b []byte

	if strings.Contains(ip, ".") {
		for _, s := range strings.Split(ip, ".") {
			i, _ := strconv.Atoi(s)
			b = append(b, byte(i))
		}
	}

	return b
}
func socks5ReadAddr(data []byte) (addr string, port uint16) {
	port = binary.BigEndian.Uint16(data[len(data)-2:])
	switch data[3] {
	case 1: //ipv4
		str := make([][]byte, 4)
		for k, v := range data[4:8] {
			str[k] = []byte(strconv.Itoa(int(v)))
		}
		addr = string(bytes.Join(str, []byte{46}))

	case 3: //域名
		addr = string(data[5 : len(data)-2])

	case 4: //ipv6
		strs := make([]string, 0)
		for i := 4; i < 20; i += 2 {
			str := ""
			for j := 0; j < 2; j++ {
				str += fmt.Sprintf("%0.2x", data[i+j])
			}
			str = strings.TrimLeft(str, "0")
			if str == "" {
				str = "0"
			}
			strs = append(strs, str)
		}
		addr = "[" + strings.Join(strs, ":") + "]"

	default:

	}

	return
}
