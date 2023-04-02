package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"math/rand"
	"net"
	"cert"
	"rakshasa/common"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/luyu6056/ishell"
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
	l.openMsg =cert.RSAEncrypterByPrivByte(append(l.randkey,[]byte(addrs[1])...))
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
				conn:   conn,
				server: n,
				randkey: l.randkey,
			}
			s.OnOpened()
			if s.connect(common.RAW_TCP, addr1.IP.String(), uint16(addr1.Port)) {
				go rawHandleLocal(s)
			} else {
				s.Close(nodeIsClose)
				if common.Debug {
					fmt.Println("Connect连接失败，远程节点已关闭")
				}
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
		if common.Debug {
			fmt.Println("发送", crc32.ChecksumIEEE(buf[8:8+n]), n)
		}
		data := make([]byte, 8+n)
		copy(data, buf)
		s.server.Write(common.CMD_CONN_MSG, s.id, buf[:8+n])
	}
}
func init() {
	bindshell := cliInit()
	bindshell.SetPrompt("rakshasa\\bind>")
	bindshell.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "列出当前连接的ID和其他信息",
		Func: func(c *ishell.Context) {
			var list []*clientListen
			currentNode.listenMap.Range(func(key, value interface{}) bool {
				if v, ok := value.(*clientListen); ok {
					if v.typ == "bind" {
						list = append(list, v)
					}
				}
				return true
			})
			orderClientListen(list)
			fmt.Println("当前连接数量:", len(list))
			for _, v := range list {
				fmt.Println("ID", v.id, "本地端口", v.localAddr, "远程端口", v.remoteAddr, "服务器uuid", v.server.uuid)
			}
		},
	})
	bindshell.AddCmd(&ishell.Cmd{
		Name: "new-bind",
		Help: "新建一个本地bind，使用方法 new-bind ip:port,remote_ip:remote_port 目标服务器  如 new-bind 192.168.1.180:8808,0.0.0.0:8808 192.168.1.2:1081",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 2 {
				c.Println("参数错误")
				return
			}

			if err := StartRawBind(c.Args[0], strings.Split(c.Args[1], ",")); err != nil {
				c.Println("启动bind失败", err)

			}
		},
	})
	bindshell.AddCmd(&ishell.Cmd{
		Name: "close",
		Help: "关闭一个bind连接，使用方法 close ID",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误，例子 close 1")
				return
			}
			id, _ := strconv.Atoi(c.Args[0])
			var l *clientListen
			if value, ok := currentNode.listenMap.Load(uint32(id)); ok {
				if v, ok := value.(*clientListen); ok && v.typ == "bind" {
					l = v
				}

			}
			if l == nil {
				c.Println("没有找到ID为", id, "的连接")
			} else {
				l.Close("命令行关闭")
				l.server.Write(common.CMD_DELETE_LISTEN, l.id, l.randkey)
				currentNode.listenMap.Delete(uint32(id))

			}
		},
	})

	rootCli.AddCmd(&ishell.Cmd{
		Name: "bind",
		Help: "进入bind功能",
		Func: func(c *ishell.Context) {
			bindshell.Run()
		},
	})
	connectshell := cliInit()
	connectshell.SetPrompt("rakshasa\\connect>")
	connectshell.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "列出当前连接的ID和其他信息",
		Func: func(c *ishell.Context) {
			var list []*clientListen
			currentNode.listenMap.Range(func(key, value interface{}) bool {
				if v, ok := value.(*clientListen); ok {
					if v.typ == "connect" {
						list = append(list, v)
					}
				}
				return true
			})
			orderClientListen(list)
			fmt.Println("当前连接数量:", len(list))
			for _, v := range list {
				fmt.Println("ID", v.id, "本地端口", v.localAddr, "远程端口", v.remoteAddr, "服务器uuid", v.server.uuid)
			}

		},
	})
	connectshell.AddCmd(&ishell.Cmd{
		Name: "new-connect",
		Help: "新建一个本地connect，使用方法 new-connect ip:port,remote_ip:remote_port 目标服务器  如 new-connect 0.0.0.0:88,192.168.1.180:8808 192.168.1.2:1081",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 2 {
				c.Println("参数错误")
				return
			}

			n, err := GetNodeFromAddrs(strings.Split(c.Args[1], ","))
			if err != nil {
				c.Println("connect连接", c.Args[1], "失败", err)
				return
			}
			if err = StartRawConnect(c.Args[0], n); err != nil {
				c.Println(err)
				return
			}
			c.Println("connect连接", c.Args[1], "成功")
		},
	})
	connectshell.AddCmd(&ishell.Cmd{
		Name: "close",
		Help: "关闭一个connect连接，使用方法 close ID",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误，例子 close 1")
				return
			}

			id, _ := strconv.Atoi(c.Args[0])
			var l *clientListen
			if value, ok := currentNode.listenMap.Load(uint32(id)); ok {
				if v, ok := value.(*clientListen); ok && v.typ == "connect" {
					l = v
				}
			}
			if l == nil {
				c.Println("没有找到ID为", id, "的连接")
			} else {
				l.Close("命令行关闭")
				l.server.Write(common.CMD_DELETE_LISTEN, l.id, l.randkey)
				currentNode.listenMap.Delete(uint32(id))

			}
		},
	})

	rootCli.AddCmd(&ishell.Cmd{
		Name: "connect",
		Help: "进入connect功能",
		Func: func(c *ishell.Context) {
			connectshell.Run()
		},
	})
}
