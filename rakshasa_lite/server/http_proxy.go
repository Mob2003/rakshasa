package server

import (
	"bufio"
	"bytes"
	"cert"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"rakshasa_lite/common"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type httpProxyClient struct {
	windowsSize int64
	status      int32
	conn        net.Conn
	udpconn     net.Conn

	remote int32
	server *node
	id     uint32
	wait   chan int
	close  string

	udpMap     sync.Map
	listenId   uint32
	localAddr  string
	method     string
	cfg        *common.Addr
	pool       *httpPool
	remoteAddr string
	remotePort string
	randkey    []byte
}

func (s *httpProxyClient) Write(b []byte) {

	switch b[0] {
	case common.CMD_CONNECT_BYIDADDR_RESULT:
		if string(s.randkey) != string(b[1:9]) {
			return
		}
		switch common.NetWork(b[9]) {

		case common.RAW_TCP:
			if b[10] != 1 {
				go func() { s.Close("") }()
			} else if s.method == "CONNECT" {
				s.conn.Write([]byte("HTTP/1.0 200 Connection established\r\n\r\n"))
			}
		case common.RAW_TCP_WITH_PROXY:

			if b[10] != 1 {
				//重新拉取一个池
				if !s.connect() {
					s.Close(nodeIsClose)
				}
			} else if s.method == "CONNECT" {
				s.conn.Write([]byte("HTTP/1.0 200 Connection established\r\n\r\n"))
			}
		default:
			log.Println("httpProxyClient 未处理")
		}

	case common.CMD_CONN_MSG:
		s.conn.Write(b[1:])
		s.Addwindow(int64(-len(b[1:])))
	default:
		log.Println("未处理")
	}

}

func (s *httpProxyClient) Close(msg string) {
	if atomic.CompareAndSwapInt32(&s.status, CONN_STATUS_CONNECT, CONN_STATUS_NONE) {

		<-s.wait
		s.wait <- common.CONN_STATUS_CLOSE

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
		if s.udpconn != nil {
			s.udpconn.Close()
		}
		s.udpMap.Range(func(k, _ interface{}) bool {
			s.udpMap.Delete(k)
			return true
		})
	}

}
func (s *httpProxyClient) Addwindow(window int64) {

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

func StartHttpProxy(cfg *common.Addr, dst []string, poolfile string) error {
	var pool *httpPool
	var err error
	if poolfile != "" {
		pool, err = httpPoolInit(poolfile)
		if err != nil {
			return err
		}
	}
	var target *node

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
		typ:       "http",
		randkey:   make([]byte, 8),
	}
	binary.LittleEndian.PutUint64(l.randkey, uint64(rand.NewSource(time.Now().UnixNano()).Int63()))
	l.listen, err = StartHttpProxyWithServer(cfg, target, l.id, pool)
	if err != nil {

		return err
	}

	currentNode.listenMap.Store(l.id, l)
	return nil
}
func StartHttpProxyWithServer(cfg *common.Addr, n *node, id uint32, pool *httpPool) (net.Listener, error) {
	l, err := net.Listen("tcp", cfg.Addr())
	if err != nil {
		return nil, err
	}
	randkey := make([]byte, 8)
	binary.LittleEndian.PutUint64(randkey, uint64(rand.NewSource(time.Now().UnixNano()).Int63()))
	fmt.Println("httpproxy start ", cfg.Addr())
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				if err.(*net.OpError).Err == net.ErrClosed {
					return
				}
				continue
			}

			s := &httpProxyClient{
				cfg:      cfg,
				conn:     conn,
				server:   n,
				listenId: id,
				pool:     pool,
				randkey:  randkey,
			}
			go handleHttpProxyLocal(s)
		}
	}()
	return l, nil
}
func (s *httpProxyClient) OnOpened() (close bool) {
	s.wait = make(chan int, 1)
	s.remote = CONN_REMOTE_OPEN
	s.windowsSize = 0
	s.wait <- common.CONN_STATUS_OK

	return
}

// 监听本地服务
func handleHttpProxyLocal(s *httpProxyClient) {
	defer func() {
		if err := recover(); err != nil {

		}
	}()
	b := make([]byte, common.MAX_PLAINTEXT-8)
	if s.OnOpened() {
		s.Close("无法获得服务器连接")
	}
	var data []byte
	var req = &http1request{}
	for {
		n, err := s.conn.Read(b)
		if err != nil {

			s.Close(err.Error())
			return
		}

		data = append(data, b[:n]...)
		//尝试读取一个http消息
		l, _, err := parsereq(req, data)
		if err != nil {
			return
		} else if l == 0 {
			continue
		}
		//判断用户名密码
		if s.cfg.GetHttpAuthorizationHeader() != "" {
			var authorize bool
			for _, herder := range req.header {

				if herder == s.cfg.GetHttpAuthorizationHeader() {
					authorize = true
					break
				}
			}
			if !authorize {
				s.conn.Write([]byte("HTTP/1.0 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"Access to internal site\"\r\nContent-Length: 0\r\n\r\n"))
				continue
			}

		}
		data = data[l:]
		switch req.method {
		case "GET":
			if u, err := url.Parse(req.uri); err == nil {
				if i := strings.IndexByte(u.Host, ':'); i > -1 {
					s.remoteAddr = u.Host[:i]
					s.remotePort = u.Host[i+1:]
				} else {
					s.remoteAddr = u.Host
					s.remotePort = "80"
				}
				if s.connect() {
					buf := bufPool.Get().(*bytes.Buffer)
					buf.Reset()
					buf.WriteString("GET ")
					buf.WriteString(req.uri)
					buf.WriteString(" HTTP/1.1\r\n")
					for _, header := range req.header {
						buf.WriteString(header)
						buf.WriteString("\r\n")
					}
					buf.WriteString("\r\n")
					s.write2connect(buf.Bytes())
					buf.Reset()
					bufPool.Put(buf)
				} else {
					s.Close(nodeIsClose)
				}

			} else {
				return
			}
		case "CONNECT":
			s.method = "CONNECT"
			if i := strings.IndexByte(req.uri, ':'); i > -1 {
				s.remoteAddr = req.uri[:i]
				s.remotePort = req.uri[i+1:]
				if !s.connect() {
					s.Close(nodeIsClose)
				}
			} else {
				return
			}

			for {
				n, err = s.conn.Read(b)
				if err != nil {
					s.Close(err.Error())
					return
				}

				s.write2connect(b[:n])
			}
		default:

		}

	}

}
func (s *httpProxyClient) write2connect(data []byte) {
	var new_size int64
	if new_size = int64(common.INIT_WINDOWS_SIZE) - s.windowsSize; new_size > 0 { //扩大窗口
		atomic.AddInt64(&s.windowsSize, new_size)

	} else {
		new_size = 0
	}
	outdata := make([]byte, 8)
	outdata[0] = byte(new_size)
	outdata[1] = byte(new_size >> 8)
	outdata[2] = byte(new_size >> 16)
	outdata[3] = byte(new_size >> 24)
	outdata[4] = byte(new_size >> 32)
	outdata[5] = byte(new_size >> 40)
	outdata[6] = byte(new_size >> 48)
	outdata[7] = byte(new_size >> 56)

	s.server.Write(common.CMD_CONN_MSG, s.id, append(outdata, data...))
}
func (s *httpProxyClient) connect() bool {
	if !s.checkConnect() {
		buf := make([]byte, 2+len(s.remoteAddr)+len(s.remotePort))
		s.id = s.server.storeConn(s)
		buf[0] = byte(common.RAW_TCP)
		copy(buf[1:], s.remoteAddr)
		buf[1+len(s.remoteAddr)] = ':'
		copy(buf[2+len(s.remoteAddr):], s.remotePort)
		//添加代理信息
		if s.pool != nil {
			proxy := s.pool.Next()
			buf[0] = byte(common.RAW_TCP_WITH_PROXY)
			buf = append(buf, []byte(" "+proxy.String())...)
		}

		s.server.Write(common.CMD_CONNECT_BYIDADDR, s.id, cert.RSAEncrypterByPrivByte(append(s.randkey, buf...)))
		if value, ok := s.server.listenMap.Load(s.listenId); ok {
			switch v := value.(type) {
			case *serverListen:
				v.connMap.Store(s.id, s)
			case *clientListen:
				v.connMap.Store(s.id, s)
			}
		}
		s.status = CONN_STATUS_CONNECT
		return true
	}
	return s.server.isClose == 0
}

// 检查一下server是否断开，尝试重连，返回是否连接
func (s *httpProxyClient) checkConnect() bool {
	if s.server.isClose == 1 {
		//尝试重连
		if newNode, _ := GetNodeFromAddrs(s.server.reConnectAddrs); newNode != nil {
			s.server = newNode
		}
	}
	return s.status == CONN_STATUS_CONNECT
}
func (s *httpProxyClient) Remoteclose() {

	s.close = "本地要求远程关闭"

	buf := make([]byte, 4)
	buf[0] = byte(s.id)
	buf[1] = byte(s.id >> 8)
	buf[2] = byte(s.id >> 16)
	buf[3] = byte(s.id >> 24)
	s.server.Write(common.CMD_DELETE_LISTENCONN_BYID, s.listenId, append(s.randkey, buf...))

}
func init() {

}

type kv struct { //kv键值对
	key   string
	value string
}
type http1request struct {
	Status string

	//解析相关
	Proto, method    string
	path, query, uri string
	keep_alive       bool
	header           []string //记录整行
	body             []byte
	//rawdata          []byte

	//输出buffer相关
	//data     io.ReadCloser  //消息主体
	//dataSize int            //dataSize大于-1就输出，所以要放到最后赋值
	//out      *tls.MsgBuffer //输出消息用buffer，包含header等信息
	//out1     *tls.MsgBuffer
	//流水线控制
	//next *http1request
	//num               int32
	//alreadyOutHreader bool
}

func (req *http1request) addheader(line string, j int) {
	if line[:j] == "Proxy-Connection" {
		req.header = append(req.header, "Connection: "+line[j+2:])
		req.keep_alive = line[j+2:] == "line[j+2:]"
	} else {
		req.header = append(req.header, line)
	}

}
func parsereq(req *http1request, data []byte) (clen int, resdata []byte, err error) {

	l := len(data)
	defer func() {
		if e := recover(); e != nil {

		}
	}()

	// method, path, proto line

	req.Proto = ""
	var s = 0
	var line string
	var firstLine = true
	req.body = req.body[:0]
	req.header = req.header[:0]
	for i, j := 0, 0; j < l; j += i + 2 {
		i = bytes.IndexByte(data[j:], 13)

		if i == -1 {
			break //跳出循环，判断是否包体过大
		}

		line = string(data[j : j+i])
		if i > 0 {
			if firstLine {
				var q = -1
				i := strings.IndexByte(line, 32)
				if i > -1 {
					req.method = line[:i]
					line = line[i+1:]
					for i, v := range line {
						if v == 63 && q == -1 {
							q = i
						} else if v == 32 {
							if q != -1 {
								req.path = line[s:q]
								req.query = line[q+1 : i]
							} else {
								req.path = line[s:i]
							}
							req.uri = line[s:i]
							i++
							req.Proto = line[i:]
							//判断http返回
							if req.method == "HTTP/1.1" || req.method == "HTTP/1.0" {
								/*code, err := strconv.Atoi(req.path)
								if err == nil {
									//req.Code = code
									//req.CodeMsg = req.Proto
								}*/
								req.Status = line
								req.Proto = req.method
								req.method = ""
								req.path = ""
							}
							break
						}
					}
				}

				switch req.Proto {
				case "HTTP/1.0":
					req.keep_alive = false
				case "HTTP/1.1":
					req.keep_alive = true
				default:
					return 0, nil, fmt.Errorf("malformed http1request")
				}
				firstLine = false
			} else {
				k := strings.IndexByte(line, 58)
				if k > -1 && k < len(line) {
					req.addheader(line, k)
				}
			}
		} else {

			j += i + 2

			if l-j < clen {
				return 0, nil, nil
			}
			req.body = append(req.body, data[j:j+clen]...)
			//req.body = append(req.body, data[s:s+clen]...)
			//req.rawdata = append(req.rawdata, data[:j+clen]...)
			return j + clen, req.body, nil

		}

	}

	return 0, nil, nil
}

type httpPool struct {
	r *bufio.Reader
	f *os.File
	sync.Mutex
}

func httpPoolInit(file string) (*httpPool, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("打开http代理池文件 %s 失败", file)
	}
	p := &httpPool{
		r:     bufio.NewReader(f),
		f:     f,
		Mutex: sync.Mutex{},
	}
	if _, err = p.do_next(0); err != nil {
		return nil, fmt.Errorf("无法从%s文件获取代理，错误%v", file, err)
	}
	return p, nil
}
func (p *httpPool) Next() *common.Addr {
	addr, _ := p.do_next(0)
	return addr
}
func (p *httpPool) do_next(n int) (*common.Addr, error) {
	if n > 100 {
		return nil, errors.New("重试错误次数过多")
	}
	p.Lock()
	line, err := p.r.ReadString(10)
	if err == io.EOF {
		p.f.Seek(0, 0)
		p.r.Reset(p.f)
		p.Unlock()
		return p.do_next(n + 1)
	}
	p.Unlock()
	line = strings.TrimRight(line, "\n")
	line = strings.TrimRight(line, "\r")

	if len(line) == 0 {
		return p.do_next(n + 1)
	}
	addr, err := common.ParseAddr(line)
	if err != nil {
		return p.do_next(n + 1)
	}
	return addr, nil
}
