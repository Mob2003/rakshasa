package server

import (
	"bytes"
	"cert"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"net/url"
	"rakshasa/aes"
	"rakshasa/common"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	bufPool = &sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
	closeChan = make(chan *bytes.Buffer, 1) //用于接收已关闭消息的黑洞chan
)

// 节点的连接，包含listen来的和主动connect的
type Conn struct {
	closeTag int32
	node     *node
	nodeaddr string
	//key              string
	remoteAddr    string
	inChan        chan func()
	OutChan       chan []byte
	close         chan string
	isClient      bool
	nodeConn      *tls.Conn
	regResult     chan error
	regResultNode chan *node
}

type serverListen struct {
	close        int32
	node         *node
	listen       net.Listener
	isSocks5     bool
	socks5Replay []byte
	replayid     uint32
	id           uint32
	connMap      sync.Map
	randkey      []byte
}
type serverConnect struct {
	close       int32
	id          uint32
	windowsSize int64
	conn        net.Conn
	node        *node
	address     string

	write chan *bytes.Buffer

	wait        chan int
	closeReason string
	randkey     []byte
}

// 中转与最终出口
func StartServer(addr string) error {
	config := cert.Tlsconfig.Clone()
	fmt.Println("start on ", addr)

	l, err := tls.Listen("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("server start fail %v", err)
	}
	currentNode.listen = l
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				if err.(*net.OpError).Err == net.ErrClosed {
					return
				}
				continue
			}

			//封装一个符合common.server接口的server

			c := &Conn{
				nodeConn:   conn.(*tls.Conn),
				remoteAddr: conn.RemoteAddr().String(),
			}
			connMap.Store(c.remoteAddr, conn)
			go c.handlerNodeRead()
			go c.handle()
		}
	}()

	return nil
}
func init() {
	go func() {
		for b := range closeChan {
			b.Reset()
			bufPool.Put(b)
		}

	}()
}

func (conn *serverConnect) Close(reason string) {
	if atomic.CompareAndSwapInt32(&conn.close, 0, 1) {
		go func() {
			if conn.conn != nil {
				conn.conn.Close()
			}
			//fmt.Println(conn.fd, reason)
			conn.node.connMap.Delete(conn.id)
			conn.closeReason = reason
			conn.node.listenMap.Range(func(key, value interface{}) bool {
				value.(*serverListen).connMap.Delete(conn.id)
				return true
			})
			if reason != remoteClose {
				conn.node.Write(common.CMD_DELETE_CONNID, conn.id, nil)
			}

			select {
			case conn.wait <- common.CONN_STATUS_CLOSE:
			case <-time.After(time.Second * 10):
			}
			conn.write <- nil
			conn.write = closeChan

		}()
	}
}
func (c *Conn) Close(reason string) {
	c.close <- reason

}

func (conn *serverConnect) handTcpReceive() {
	go func() {
		for b := range conn.write {
			if b == nil {
				conn.write = closeChan
				return
			}
			if _, err := conn.conn.Write(b.Bytes()); err != nil {
				conn.Close(err.Error())
			}
			b.Reset()
			bufPool.Put(b)
		}
	}()
	var err error
	var n int
	defer func() {

		if err != nil {
			conn.Close(conn.address + " 读取出错" + err.Error())
		} else {
			conn.Close(conn.address + " read异常关闭")
		}

	}()

	buf := make([]byte, common.MAX_PLAINTEXT)

	for conn.close == 0 {
		conn.conn.SetReadDeadline(time.Now().Add(common.WRITE_DEADLINE))
		n, err = conn.conn.Read(buf)
		if err != nil {
			if atomic.LoadInt32(&conn.close) == 0 {
				if e := err.Error(); !strings.Contains(e, ": i/o timeout") {

					return
				}
				continue
			} else {
				return
			}
		}
		data := make([]byte, n)
		copy(data, buf)
		if common.Debug {

			fmt.Println("发送", crc32.ChecksumIEEE(data), n)
		}
		conn.node.Write(common.CMD_CONN_MSG, conn.id, data)

		atomic.AddInt64(&conn.windowsSize, -1*int64(n))

		for atomic.LoadInt64(&conn.windowsSize) <= 0 && conn.close == 0 {

			select {
			case flag := <-conn.wait:
				if flag == common.CONN_STATUS_CLOSE {
					return
				}
			case <-time.After(time.Second):
			}
		}
	}

}
func (conn *serverConnect) Write(data []byte) {
	data = data[1:]
	windows_update_size := int64(data[0]) | int64(data[1])<<8 | int64(data[2])<<16 | int64(data[3])<<24 | int64(data[4])<<32 | int64(data[5])<<40 | int64(data[6])<<48 | int64(data[7])<<56

	if windows_update_size != 0 {

		old := atomic.AddInt64(&conn.windowsSize, windows_update_size) - windows_update_size
		if old < 0 {
			go func() {
				select {
				case conn.wait <- common.CONN_STATUS_OK:
				case <-time.After(time.Second):
				}
			}()
		}
	}
	if common.Debug {

		fmt.Println("收到", crc32.ChecksumIEEE(data[8:]), len(data[8:]))
	}
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()
	b.Write(data[8:])
	conn.write <- b
}
func (conn *serverConnect) handUdpReceive() {

	var err error
	var n int
	defer func() {

		if err != nil {
			conn.Close(conn.address + " 网站读取出错" + err.Error())
		} else {
			conn.Close(conn.address + " read异常关闭")
		}

	}()

	buf := make([]byte, common.MAX_PLAINTEXT)

	for conn.close == 0 {
		conn.conn.SetReadDeadline(time.Now().Add(common.WRITE_DEADLINE))
		n, err = conn.conn.Read(buf)
		if err != nil {
			if atomic.LoadInt32(&conn.close) == 0 {
				if e := err.Error(); !strings.Contains(e, ": i/o timeout") {

					return
				}
				continue
			} else {
				return
			}
		}

		b := make([]byte, n)
		copy(b, buf)
		conn.node.Write(common.CMD_CONN_UDP_MSG, conn.id, b)

	}

}

func (conn *serverConnect) doConnectTcp(network common.NetWork, addr string) {

	netconn, err := net.DialTimeout("tcp", addr, time.Second*30)
	if err != nil {
		buf := make([]byte, 2)
		buf[0] = byte(network)
		buf[1] = 0
		conn.node.Write(common.CMD_CONNECT_BYIDADDR_RESULT, conn.id, append(conn.randkey, buf...))
		conn.Close("fd拨号失败")
		return
	} else {
		buf := make([]byte, 2)
		buf[0] = byte(network)
		buf[1] = 1
		conn.node.Write(common.CMD_CONNECT_BYIDADDR_RESULT, conn.id, append(conn.randkey, buf...))
		if conn.close == 0 {
			conn.conn = netconn
			go conn.handTcpReceive()

		}

	}
}
func (conn *serverConnect) doConnectTcpWithHttpProxy(network common.NetWork, addr string) {
	writeResult := func(res bool) {
		buf := make([]byte, 2)
		buf[0] = byte(network)
		buf[1] = 0
		if res {
			buf[1] = 1
		}
		conn.node.Write(common.CMD_CONNECT_BYIDADDR_RESULT, conn.id, append(conn.randkey, buf...))
	}

	if i := strings.IndexByte(addr, 32); i > -1 {
		cfg, err := common.ParseAddr(addr[i+1:])
		if err != nil {
			writeResult(false)
			conn.Close("地址解析失败")
			return
		}
		netconn, err := net.DialTimeout("tcp", cfg.Addr(), time.Second*2)
		if err != nil {
			writeResult(false)
			conn.Close("fd拨号失败")
			return
		} else {
			netconn.SetDeadline(time.Now().Add(time.Second * 30))
			netconn.SetWriteDeadline(time.Now().Add(time.Second * 30))
			switch cfg.Scheam() {
			case "", "http://":
				//请求代理
				data := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Connection: keep-alive\r\nUser-Agent: Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36\r\n", addr[:i], addr[:i])
				if cfg.GetHttpAuthorizationHeader() != "" {
					data += cfg.GetHttpAuthorizationHeader() + "\r\n\r\n"
				} else {
					data += "\r\n"
				}
				_, err = netconn.Write([]byte(data))
				if err != nil {
					writeResult(false)
					conn.Close("http代理发送消息失败")
					return
				}
				var resdata []byte
				var result [8192]byte
				var req = &http1request{}
				for {
					n, err := netconn.Read(result[:])
					if err != nil {
						writeResult(false)
						conn.Close("读取http代理结果失败")
						return
					}

					resdata = append(resdata, result[:n]...)
					l, _, err := parsereq(req, resdata)
					if err != nil {
						return
					} else if l > 0 {
						break
					}

				}
				if req.Status == "200 Connection established" {
					writeResult(true)
					if conn.close == 0 {
						conn.conn = netconn
						go conn.handTcpReceive()

					}
				} else {
					writeResult(false)
					conn.Close("http代理连接失败")
				}
				writeResult(true)
			case "socks5://":
				_, err = netconn.Write([]byte{5, 1, 2})
				if err != nil {
					writeResult(false)
					conn.Close("socks5代理发送消息失败")
					return
				}
				var result [8192]byte

				n, err := netconn.Read(result[:])
				if err != nil {
					writeResult(false)
					conn.Close("读取socks5数据出错")
					return
				}

				if string(result[:n]) == string([]byte{5, 2}) { //需要认证
					user, password := cfg.User(), cfg.Password()
					if user == "" && password == "" {
						writeResult(false)
						conn.Close("socks5需要验证")
						return
					}

					data := make([]byte, (3 + len(user) + len(password)))
					data[0] = 5
					data[1] = byte(len(user))
					copy(data[2:], user)
					data[2+len(user)] = byte(len(password))
					copy(data[3+len(user):], password)
					netconn.Write(data)
					n, err = netconn.Read(result[:])
					if err != nil || string(result[:n]) != string([]byte{5, 0}) {
						writeResult(false)
						conn.Close("密码校验不通过")
						return
					}
				}

				data := []byte{5, 1, 0, 1}
				if u, err := url.ParseRequestURI(addr[:i]); err == nil {
					data[3] = 3
					data = append(data, byte(len(u.Scheme)))
					data = append(data, u.Scheme...)
					p, _ := strconv.Atoi(u.Opaque)
					port := []byte{byte(p >> 8), byte(p)}
					data = append(data, port...)
				} else if tcp4, err := net.ResolveTCPAddr("tcp4", addr[:i]); err == nil {
					data[3] = 1
					data = append(data, tcp4.IP.String()...)
					port := []byte{byte(tcp4.Port), byte(tcp4.Port >> 8)}
					data = append(data, port...)
				} else if tcp6, err := net.ResolveTCPAddr("tcp6", addr[:i]); err == nil {
					data[3] = 4
					data = append(data, tcp6.IP.String()...)
					port := []byte{byte(tcp6.Port), byte(tcp6.Port >> 8)}
					data = append(data, port...)
				}
				netconn.Write(data)
				n, _ = netconn.Read(result[:])
				if n >= 2 && string(result[:2]) == string([]byte{5, 0}) {
					writeResult(true)
					if conn.close == 0 {
						conn.conn = netconn
						go conn.handTcpReceive()

					}
				} else {
					writeResult(false)
					conn.Close("socks5连接失败")
				}

			default:
				writeResult(false)
				conn.Close("不支持的代理协议")
			}

		}
	} else {
		writeResult(false)
		conn.Close("无法获取代理地址")

	}

}

func (conn *serverConnect) doHandleUdp() {
	for b := range conn.write {
		if b == nil {
			conn.write = closeChan
			return
		}

		conn.conn.Write(b.Bytes())
		b.Reset()
		bufPool.Put(b)
	}
}

var broadcastMap sync.Map //广播帧防止重复处理
func (c *Conn) handlerNodeRead() {
	var err error
	defer func() {
		c.nodeConn.Close()
		c.Close("read错误" + err.Error())

	}()

	lengbuf := make([]byte, 2)
	for {
		_, err = io.ReadFull(c.nodeConn, lengbuf)
		if err != nil {
			if strings.Contains(err.Error(), "i/o timeout") {
				continue
			}
			return
		}

		buf := make([]byte, int(lengbuf[0])+int(lengbuf[1])<<8)
		_, err = io.ReadFull(c.nodeConn, buf)
		b := aes.AesCtrDecrypt(buf)
		msg := common.UnmarshalMsg(b)
		if common.Debug {
			fmt.Println("fromto", msg.From, msg.To, common.CmdToName[msg.CmdOpteion], int(lengbuf[0])+int(lengbuf[1])<<8)
		}

		if msg.To == common.NoneUUID.String() && c.node == nil {
			c.inChan <- func() {
				newNode := &node{
					conn: c,
				}
				newNode.do(msg)
			}
		} else if msg.To == currentNode.uuid {

			func() {
				l := clientLock.RLock()
				v, ok := nodeMap[msg.From]
				l.RUnlock()
				if ok && v.port != 0 {
					c.inChan <- func() {
						v.do(msg)
					}
				} else {
					l := clientLock.Lock()
					v, ok := nodeMap[msg.From]
					if !ok {
						newNode := &node{
							uuid:    msg.From,
							conn:    c,
							waitMsg: []*common.Msg{msg},
						}
						result := make(chan interface{}, 1)
						id := newNode.storeQuery(result)
						if common.Debug {
							fmt.Printf("nodeMap1 %s %p \r\n", msg.From, newNode)
						}
						nodeMap[msg.From] = newNode
						l.Unlock()
						newNode.Write(common.CMD_GET_CURRENT_NODE, id, []byte{1}) //获取丢失节点的信息
						go func() {
							defer newNode.deleteQuery(id)
							select {
							case res := <-result:
								if res == nil {

									for _, m := range newNode.waitMsg {
										c.inChan <- func() {
											newNode.do(m)
										}
									}
								}
							case <-time.After(common.CMD_TIMEOUT):
								newNode.Delete("超时")
							}
						}()

					} else {

						if msg.CmdOpteion == common.CMD_GET_CURRENT_NODE_RESULT {

							var res chan interface{}
							if _v, ok := v.loadQuery(msg.CmdId); !ok {
								return
							} else {
								res = _v
							}

							var nmsg nodeInfo
							err = json.Unmarshal(msg.CmdData, &nmsg)
							if err != nil {
								res <- err
								return
							}
							v.hostName = cert.RSADecrypterStr(nmsg.HostName)
							v.uuid = cert.RSADecrypterStr(nmsg.UUID)
							if v.port, err = strconv.Atoi(cert.RSADecrypterStr(nmsg.Port)); err != nil {
								v.port = -1
							}
							v.mainIp = cert.RSADecrypterStr(nmsg.MainIp)
							v.goos = cert.RSADecrypterStr(nmsg.Goos)
							res <- nil
						} else {
							v.waitMsg = append(v.waitMsg, msg)
						}

						l.Unlock()
					}

				}
			}()

		} else {

			key := msg.From + "_" + strconv.Itoa(int(msg.MsgId))
			msg.Ttl++
			if _, ok := broadcastMap.LoadOrStore(key, struct{}{}); !ok {

				if msg.From != currentNode.uuid && msg.To == common.BroadcastUUID.String() && msg.Ttl < 250 { //广播
					go allNodesDo(func(_n *node) (bool, error) {
						if _n.uuid != currentNode.uuid {
							_n.WriteMsg(msg)
						}
						return true, nil
					})
					if common.Debug {
						fmt.Println("广播do")
					}

					newNode := &node{
						conn: c,
					}
					c.inChan <- func() {
						newNode.do(msg)
					}
				} else {
					c.WriteToUUID(msg)
				}
				time.AfterFunc(time.Hour, func() {
					broadcastMap.Delete(key)
				})

			}

		}

	}
}

func (c *Conn) handle() {
	c.OutChan = make(chan []byte, 64)
	c.inChan = make(chan func())

	c.close = make(chan string, 999)

	go func() {
		for {
			select {
			case f := <-c.inChan:
				f()
			//c.do(b)
			case b := <-c.OutChan:

				if c.closeTag == 0 {
					c.tlsWrite(b)
					//var err error
					for i := 0; i < len(c.OutChan); i++ {

						c.tlsWrite(<-c.OutChan)
					}
				}

			case reason := <-c.close:
				c.OutChan = upNodeWrite
				if c.node != nil && c.node.nextPingTime > time.Now().Unix()+5 {
					c.node.ping(0)
					c.node.nextPingTime = time.Now().Unix() + 5
				}
				func() { //返回false则退出handle
					connMap.Delete(c.remoteAddr)
					l := clientLock.Lock()
					defer func() {
						l.Unlock()
					}()

					if atomic.CompareAndSwapInt32(&c.closeTag, 0, 1) {
						if common.Debug {
							fmt.Println(c.nodeConn.RemoteAddr().String(), "关闭原因", reason)
						}
						if c.nodeConn != nil {
							if common.Debug {
								fmt.Println("執行close1")
							}
							c.nodeConn.Close()
						}

						if c.node != nil {
							//移除上游连接
							for i := len(upLevelNode) - 1; i >= 0; i-- {
								n := upLevelNode[i]
								if n.uuid == c.node.uuid {
									upLevelNode = append(upLevelNode[:i], upLevelNode[i+1:]...)
								}
							}
						}

					}

					return
				}()
				return
			}
		}
	}()

}
func (c *Conn) reg() error {

	var err error
	reg := &common.RegMsg{
		UUID:     currentNode.uuid,
		MainIp:   cert.RSAEncrypterStr(currentNode.mainIp),
		Port:     cert.RSAEncrypterStr(strconv.Itoa(currentNode.port)),
		Goos:     cert.RSAEncrypterStr(currentNode.goos),
		Hostname: cert.RSAEncrypterStr(currentNode.hostName),
	}
	regb, _ := json.Marshal(reg)
	msg := common.Msg{
		From:       currentNode.uuid,
		To:         common.NoneUUID.String(),
		CmdOpteion: common.CMD_REG,
		CmdData:    regb,
	}
	if err = c.tlsWrite(msg.Marshal()); err != nil {
		return err
	}
	go c.handlerNodeRead()
	return nil
}
func (c *Conn) WriteToUUID(msg *common.Msg) {

	l := clientLock.RLock()
	defer l.RUnlock()

	if n, ok := nodeMap[msg.To]; ok {
		n.WriteMsg(msg)
	}
}

func (c *Conn) Write(b []byte) {

	c.OutChan <- b
}

func (c *Conn) tlsWrite(b []byte) error {
	c.nodeConn.SetWriteDeadline(time.Now().Add(common.WRITE_DEADLINE))
	n, err := c.nodeConn.Write(b)
	if common.Debug {
		if c.node!=nil{
			fmt.Println("writeto", c.node.uuid, n)
		}else{
			fmt.Println("writeto",common.NoneUUID, n)
		}

	}
	if err != nil {
		c.Close("Write " + err.Error())
		upNodeWrite <- b
	}
	return err
}
