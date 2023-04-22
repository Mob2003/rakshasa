package server

import (
	"bytes"
	"cert"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"rakshasa/aes"
	"rakshasa/common"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/abiosoft/readline"
	"github.com/google/uuid"
	"github.com/luyu6056/ishell"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var rootCli = cliInit()

func CliRun() {
	rootCli.Run()
}
func init() {
	rootCli.SetPrompt("rakshasa>")
}
func cliInit() *ishell.Shell {
	shell := ishell.New()
	shell.AddCmd(&ishell.Cmd{
		Name: "ping",
		Help: "ping 节点",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误，使用方法 ping 服务器地址")
				return
			}
			var err error
			n, err := getNode(c.Args[0])
			if err != nil {
				c.Println("无法连接节点", c.Args[0])
				return
			}

			be := time.Now()

			resChan := make(chan interface{}, 1)
			id := n.storeQuery(resChan)
			go n.ping(id)
			select {
			case <-resChan:
				n.deleteQuery(id)
				c.Println("ping", n.uuid, time.Since(be))
			case <-time.After(common.CMD_TIMEOUT):
				n.deleteQuery(id)
				c.Println("ping time out")
			}
		},
	})
	shell.AddCmd(&ishell.Cmd{
		Name: "new",
		Help: "与一个或者多个节点连接，使用方法 new ip:端口 多个地址以,间隔 如1080 127.0.0.1:1081,127.0.0.1:1082",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误，使用方法 connect ip:端口")
				return
			}
			for _, addr := range strings.Split(c.Args[0], ",") {
				_, err := getNode(addr)
				if err != nil {
					c.Println("连接", addr, "失败", err)
					return
				}
			}
		},
	})
	shell.AddCmd(&ishell.Cmd{
		Name: "print",

		Help: "列出所有节点",
		Func: func(c *ishell.Context) {

			printNodes(c)
		},
	})
	if common.Debug {
		shell.AddCmd(&ishell.Cmd{
			Name: "printConn",
			Help: "列出所有链接",
			Func: func(c *ishell.Context) {
				printConn()
			},
		})
		shell.AddCmd(&ishell.Cmd{
			Name: "printLock",
			Help: "列出所有锁",
			Func: func(c *ishell.Context) {
				printLock()
			},
		})
		shell.AddCmd(&ishell.Cmd{
			Name: "delete",
			Help: "sync.Map删除一个node ID",
			Func: func(c *ishell.Context) {
				l := clientLock.Lock()
				defer l.Unlock()
				if len(c.Args) != 1 {
					c.Println("参数不对")
					return
				}
				n, ok := nodeMap.Load(c.Args[0])
				if ok {
					n.(*node).Delete("")
				}

			},
		})
		shell.AddCmd(&ishell.Cmd{
			Name: "closenode",
			Help: "关闭一个node ID",
			Func: func(c *ishell.Context) {
				l := clientLock.Lock()
				defer l.Unlock()
				if len(c.Args) != 1 {
					c.Println("参数不对")
					return
				}
				n, ok := nodeMap.Load(c.Args[0])
				if ok {
					n.(*node).Close("debug关闭")
				}

			},
		})
	}
	return shell
}

func init() {

	configShell := cliInit()
	configShell.SetPrompt("rakshasa\\config>")
	configShell.AddCmd(&ishell.Cmd{
		Name: "info",
		Help: "打印当前配置",
		Func: func(c *ishell.Context) {
			c.Println("当前节点", currentNode.uuid)
			c.Println("上级节点地址", currentConfig.DstNode)
			c.Println("通讯密码", currentConfig.Password)
			c.Println("监听端口", currentConfig.Port)
			c.Println("监听IP", currentConfig.ListenIp)
			c.Println("禁止额外连接", currentConfig.Limit)
			c.Println("配置文件名", currentConfig.FileName)
			if currentConfig.FileSave {
				c.Println("当前配置：已写入文件")
			} else {
				c.Println("当前配置：未写入文件")
			}
		},
	})
	configShell.AddCmd(&ishell.Cmd{
		Name: "save",
		Help: "保存文件",
		Func: func(c *ishell.Context) {
			if err := ConfigSave(); err == nil {
				c.Println("写入成功")
			} else {
				c.Println("保存失败", err.Error())
			}
		},
	})
	configShell.AddCmd(&ishell.Cmd{
		Name: "d",
		Help: "修改上级节点地址，格式为 ip:端口 多个节点以,隔开 注意：不会立刻连接设置节点， 当发生 节点掉线重连 时候会连接该地址",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误，格式为 ip:端口 多个节点以,隔开 如 d 192.168.1.1:8883,192.168.1.2:8883")
				return
			}
			dstNode, err := common.ResolveTCPAddr(c.Args[0])
			if err != nil {
				c.Println("参数错误，格式为 ip:端口 多个节点以,隔开 如 d 192.168.1.1:8883,192.168.1.2:8883")
				return
			}
			currentConfig.DstNode = dstNode
			currentConfig.FileSave = false
		},
	})
	configShell.AddCmd(&ishell.Cmd{
		Name: "password",
		Help: "修改通讯密码,立即生效",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误，格式为 password \"123456\"")
				return
			}
			c.Println(c.Args)
			currentConfig.Password = c.Args[0]
			currentConfig.FileSave = false
			aes.Key = aes.MD5_B(currentConfig.Password + string(cert.RsaPrivateKey[:16]))
		},
	})
	configShell.AddCmd(&ishell.Cmd{
		Name: "port",
		Help: "修改监听端口,立即生效",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误，格式为 port 8883")
				return
			}
			port, _ := strconv.Atoi(c.Args[0])
			if port <= 0 || port > 65535 {
				c.Println("参数错误，端口范围是1-65535")
				return
			}
			c.Println("正在关闭server监听")
			if currentNode.listen != nil {
				currentNode.listen.Close()
				currentNode.listen = nil
			}
			currentConfig.Port = port
			currentNode.port = port
			currentConfig.FileSave = false
			if err := StartServer(fmt.Sprintf(":%d", currentConfig.Port)); err != nil {
				c.Printf("启动节点失败 %v, 请重新修改监听端口", currentConfig.Port)
			}
		},
	})

	configShell.AddCmd(&ishell.Cmd{
		Name: "ip",
		Help: "修改本节点连接ip，当其他节点进行额外连接时候，优先使用此ip连接",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误")
				return
			}

			currentConfig.ListenIp = c.Args[0]
			currentNode.mainIp = currentConfig.ListenIp
			currentConfig.FileSave = false

		},
	})
	configShell.AddCmd(&ishell.Cmd{
		Name: "limit",
		Help: "修改本节点Limit设置，使用方法 limit true",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误")
				return
			}
			currentConfig.Limit = c.Args[0] == "true"
			currentConfig.FileSave = false
		},
	})
	configShell.AddCmd(&ishell.Cmd{
		Name: "f",
		Help: "修改配置文件名，使用方法 f config.yaml",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误")
				return
			}
			currentConfig.FileName = c.Args[0]
			currentConfig.FileSave = false
		},
	})
	configShell.AddCmd(&ishell.Cmd{
		Name: "uuid",
		Help: "修改本节点UUID设置，使用方法uuid 字串符",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误")
				return
			}
			if id, err := uuid.Parse(c.Args[0]); err == nil {
				nodeMap.Delete(currentConfig.UUID)
				currentConfig.UUID = id.String()
				nodeMap.Store(currentConfig.UUID, currentNode)
				currentConfig.FileSave = false
				SetConfig(currentConfig)
			} else {
				c.Println("输入的uuid不是合法的uuid，建议使用xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
			}

		},
	})
	rootCli.AddCmd(&ishell.Cmd{
		Name: "config",
		Help: "配置管理",
		Func: func(c *ishell.Context) {
			configShell.Run()
		},
	})
	remoteShell := cliInit()

	remoteShell.SetPrompt("rakshasa\\remoteshell>")

	fileShell := cliInit()
	remoteShell.AddCmd(&ishell.Cmd{
		Name: "file",
		Help: "连到节点进行文件管理，参数为id或者uuid",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误")
				return
			}

			workN, _ := getNode(c.Args[0])
			if workN == nil {
				c.Println("无法连接节点", c.Args[0])
				return
			}

			if workN != nil {
				fileShell.Set("node", workN)
				result := make(chan interface{}, 1)
				id := workN.storeQuery(result)
				workN.Write(common.CMD_PWD, id, []byte(cert.RSAEncrypterByPriv(currentConfig.Password)))
				select {
				case pwd := <-result:
					workN.deleteQuery(id)
					pwd = strings.ReplaceAll(pwd.(string), "\\", "/")
					fileShell.Set("pwd", pwd)
					fileShell.SetPrompt(workN.uuid + " " + pwd.(string) + ">")
					fileShell.Run()
				case <-time.After(common.CMD_TIMEOUT):
					workN.deleteQuery(id)
					c.Println("连接", c.Args[0], "超时")
				}

			}

		},
	})
	fileShell.AddCmd(&ishell.Cmd{
		Name: "dir",
		Help: "打印当前目录文件",
		Func: func(c *ishell.Context) {
			pwd := fileShell.Get("pwd")

			n := c.Get("node").(*node)
			resChan := make(chan interface{}, 1)
			id := n.storeQuery(resChan)
			n.Write(common.CMD_DIR, id, []byte(cert.RSAEncrypterByPriv(pwd.(string))))
			select {
			case res := <-resChan:
				n.deleteQuery(id)
				c.Println(res)
			case <-time.After(common.CMD_TIMEOUT):
				n.deleteQuery(id)
				c.Println("dir time out")
			}
		},
	})
	fileShell.AddCmd(&ishell.Cmd{
		Name: "cd",
		Help: "切换工作目录",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误")
				return
			}
			dir := c.Args[0]
			pwd := fileShell.Get("pwd").(string)
			n := c.Get("node").(*node)

			if strings.Contains(dir, ":/") || dir[0] == '/' || dir == "~" {
				pwd = dir
			} else {
				pwd += "/" + dir
				pwd = strings.TrimRight(getRealPath(pwd), "/")
			}

			resChan := make(chan interface{}, 1)
			id := n.storeQuery(resChan)
			n.Write(common.CMD_CD, id, []byte(cert.RSAEncrypterByPriv(pwd)))

			select {
			case res := <-resChan:
				n.deleteQuery(id)
				if err, ok := res.(error); ok {
					c.Println(err.Error())
				} else {
					pwd = res.(string)
					fileShell.Set("pwd", pwd)
					c.SetPrompt(n.uuid + " " + pwd + ">")
				}

			case <-time.After(common.CMD_TIMEOUT):
				n.deleteQuery(id)
				c.Println("dir time out")
			}
		},
	})
	fileShell.AddCmd(&ishell.Cmd{
		Name: "upload",
		Help: "上传文件 ，upload 本地文件 远程目录(为空传到工作目录)",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 && len(c.Args) != 2 {
				c.Println("参数错误")
				return
			}
			s, err := os.Stat(c.Args[0])
			if err != nil {
				c.Println("打开本地文件", c.Args[0], "错误 ", err)
				return
			}
			f, err := os.Open(c.Args[0])
			if err != nil {
				c.Println("打开本地文件", c.Args[0], "错误 ", err)
				return
			}
			defer f.Close()
			pwd := fileShell.Get("pwd").(string) + "/"
			n := c.Get("node").(*node)

			if len(c.Args) == 2 {
				pwd = c.Args[1]
			}
			pwd = strings.ReplaceAll(pwd, "\\", "/")
			c.Args[0] = strings.ReplaceAll(c.Args[0], "\\", "/")
			i := strings.LastIndex(c.Args[0], "/")
			if i == -1 {
				i = 0
			}

			if pwd[len(pwd)-1] == '/' {
				pwd += c.Args[0][i:]
			}
			i = strings.LastIndex(pwd, "/")
			if i == -1 {
				i = 0
			}
			filename := pwd[i+1:]
			dir := pwd[:i]
			dir = strings.TrimRight(getRealPath(dir), "/") + "/"
			pwd = dir + filename
			resChan := make(chan interface{}, 9999) //避免收消息阻塞

			filereadChan := make(chan []byte, 10)

			upload := func() {
				for i := 0; i < 10; i++ {
					buf := make([]byte, common.MAX_PACKAGE-len(pwd)-9)
					n, err := f.Read(buf)
					if err != nil {
						if err == io.EOF {

							return
						}
						resChan <- err
						c.Println("读取文件", c.Args[0], "错误", err)
						return
					}

					filereadChan <- buf[:n]
				}
			}

			offset := 0
			be := len(pwd) + 1
			id := n.storeQuery(resChan)
			defer n.deleteQuery(id)
			b := []byte(pwd)
			b = append(b, 0, 0, 0, 0, 0, 0, 0, 0, 0)
			c.ProgressBar().Start()
			go upload()
			var resnum int
			for {
				select {
				case data := <-filereadChan:

					b[be] = byte(offset)
					b[be+1] = byte(offset >> 8)
					b[be+2] = byte(offset >> 16)
					b[be+3] = byte(offset >> 24)
					b[be+4] = byte(offset >> 32)
					b[be+5] = byte(offset >> 40)
					b[be+6] = byte(offset >> 48)
					b[be+7] = byte(offset >> 56)
					offset += len(data)
					n.Write(common.CMD_UPLOAD, id, cert.RSAEncrypterByPrivByte(append(b, data...)))
				case res := <-resChan:
					switch v := res.(type) {
					case error:
						c.ProgressBar().Stop()
						c.Println("上传失败", res)
						return
					case int64:
						resnum++
						i := v * 100 / s.Size()
						c.ProgressBar().Suffix(fmt.Sprint(" ", i, "%"))
						c.ProgressBar().Progress(int(i))
						if v == s.Size() {
							c.ProgressBar().Stop()
							c.Println(c.Args[0], "上传成功")
							return
						}
						if resnum >= 5 {
							go upload()
							resnum -= 10
						}
					default:
						c.Println("协议错误")
						return
					}

				case <-time.After(common.CMD_TIMEOUT):
					c.ProgressBar().Stop()
					c.Println("upload time out")
					return
				}
			}

		},
	})
	fileShell.AddCmd(&ishell.Cmd{
		Name: "download",
		Help: "下载文件 ，download 远程文件 本地目录(为空本地执行目录)",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 && len(c.Args) != 2 {
				c.Println("参数错误")
				return
			}
			pwd := fileShell.Get("pwd").(string)
			n := c.Get("node").(*node)
			file := c.Args[0]
			file = strings.ReplaceAll(file, "\\", "/")

			if strings.Contains(file, ":/") || file[0] == '/' {
				pwd = file
			} else {
				pwd += "/" + file

			}
			i := strings.LastIndex(pwd, "/")
			if i == -1 {
				i = 0
			}
			filename := pwd[i+1:]
			dir := pwd[:i]
			dir = strings.TrimRight(getRealPath(dir), "/") + "/"
			mydir, err := os.Getwd()
			local := "./" + filename
			if err == nil {
				local = mydir + "/" + filename
			}

			if len(c.Args) == 2 {
				s, err := os.Stat(c.Args[1])
				if err == nil {
					if s.IsDir() {
						local = strings.TrimRight(c.Args[1], "/") + "/" + filename
					} else {
						local = c.Args[1]
					}
				} else {
					local = c.Args[1]
				}
			}
			pwd = dir + filename

			result := make(chan interface{}, 999)
			id := n.storeQuery(result)
			defer n.deleteQuery(id)
			b := []byte(pwd)
			b = append(b, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0}...)
			total := int64(-1)
			be := len(pwd) + 1
			b[be] = byte(total)
			b[be+1] = byte(total >> 8)
			b[be+2] = byte(total >> 16)
			b[be+3] = byte(total >> 24)
			b[be+4] = byte(total >> 32)
			b[be+5] = byte(total >> 40)
			b[be+6] = byte(total >> 48)
			b[be+7] = byte(total >> 56)
			n.Write(common.CMD_DOWNLOAD, id, cert.RSAEncrypterByPrivByte(b))
			c.ProgressBar().Start()
			size := int64(0)
			resnum := 0
			total = 0
			var f *os.File
			for {
				select {
				case res := <-result:
					switch v := res.(type) {
					case error:
						c.ProgressBar().Stop()
						c.Println("下载失败", res)
						return
					case int64:
						var err error
						size = v
						f, err = os.OpenFile(local, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
						if err != nil {
							c.Println("本地文件 ", local, "写入失败", err.Error())
							return
						}
						defer f.Close()
					case []byte:
						if f == nil {
							c.Println("本地文件 ", local, "不可写入")
							return
						}
						resnum++
						num, err := f.Write(v)
						if err != nil {
							c.Println("本地文件 ", local, "写入失败", err.Error())
							return
						}
						if num != len(v) {
							c.Println("本地文件 ", local, "写入失败,写入量不符")
							return
						}
						total += int64(num)
						i := total * 100 / size
						c.ProgressBar().Suffix(fmt.Sprint(" ", i, "%"))
						c.ProgressBar().Progress(int(i))
						if total == size {
							c.ProgressBar().Stop()
							c.Println(c.Args[0], "下载成功 文件保存到", local)
							return
						}
						if resnum == 10 {
							resnum -= 10
							b[be] = byte(total)
							b[be+1] = byte(total >> 8)
							b[be+2] = byte(total >> 16)
							b[be+3] = byte(total >> 24)
							b[be+4] = byte(total >> 32)
							b[be+5] = byte(total >> 40)
							b[be+6] = byte(total >> 48)
							b[be+7] = byte(total >> 56)
							n.Write(common.CMD_DOWNLOAD, id, cert.RSAEncrypterByPrivByte(b))
						}
					default:
						c.Println("协议错误")
						return
					}

				case <-time.After(common.CMD_TIMEOUT):
					c.ProgressBar().Stop()
					c.Println("upload time out")
					return
				}
			}
		},
	})

	remoteShell.AddCmd(&ishell.Cmd{
		Name: "new",
		Help: "与一个或者多个节点连接，使用方法 new ip:端口 多个地址以,间隔 如1080 127.0.0.1:1081,127.0.0.1:1082",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误，使用方法 connect ip:端口")
				return
			}
			for _, addr := range strings.Split(c.Args[0], ",") {
				_, err := getNode(addr)
				if err != nil {
					c.Println("连接", addr, "失败", err)
					return
				}
			}
		},
	})
	remoteShell.AddCmd(&ishell.Cmd{
		Name: "shell",
		Help: "反弹shell   使用方法 shell id/uuid 启动参数 ，启动参数可为空，win默认启动cmd，linux默认启动bash， 如 shell 1 powershell 。 shell 1 zsh",
		Func: func(c *ishell.Context) {
			if len(c.Args) < 1 {
				c.Println("参数错误，例子 shell 1 powershell")
				return
			}
			param := ""
			if len(c.Args) == 2 {
				param = c.Args[1]
			}
			n, _ := getNode(c.Args[0])
			if n == nil {
				c.Println("无法连接节点", c.Args[0])
				return
			}
			res := make(chan interface{}, 999)
			id := n.storeQuery(res)

			defer n.deleteQuery(id)
			p := StartCmdParam{
				Param: param,
				Size:  common.GetSize(),
			}

			b, _ := json.Marshal(p)
			n.Write(common.CMD_SHELL, id, cert.RSAEncrypterByPrivByte(b))
			s := &remoteCmd{
				cmd:       nil,
				stdin:     nil,
				inChan:    make(chan []byte, 999),
				translate: func(in []byte) ([]byte, error) { return in, nil },
				pong:      time.Now().Unix(),
			}

			select {
			case i := <-res:
				switch v := i.(type) {
				case error:
					c.Println("启动shell失败，错误", v.Error())
				case []byte:
					data := v

					s.id = uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
					switch data[4] {
					case 0: //windows
						if string(data[len(data)-6:]) == string([]byte{32, 57, 51, 54, 13, 10}) { //活动代码页: 936
							//gbk转utf8
							s.translate = func(in []byte) ([]byte, error) {
								reader := transform.NewReader(bytes.NewReader(in), simplifiedchinese.GBK.NewDecoder())
								d, e := ioutil.ReadAll(reader)
								if e != nil {
									return nil, e
								}
								return d, nil
							}
						}
					case 1: //linux
						if runtime.GOOS == "windows" {
							if !common.EnableTermVt {
								s.translate = func(in []byte) ([]byte, error) {
									if in[0] == 27 {
										r, _ := regexp.Compile(`\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])`)
										res := r.ReplaceAllString(string(in), "")
										return []byte(res), nil
									}
									return in, nil

								}
							}

						}

					}
					atomic.CompareAndSwapInt32(&s.cmdStatus, 0, 1)
				}
			case <-time.After(common.CMD_TIMEOUT):
				c.Println("启动shell失败，超时")
				return
			}

			n.shellMap.Store(s.id, s)
			r, _ := readline.NewEx(&readline.Config{FuncIsTerminal: func() bool { return false }, ForcePrint: true})
			defer func() {
				n.shellMap.Delete(s.id)
				atomic.StoreInt32(&s.cmdStatus, -1)
				c.Println("请按回车键退出")
				r.Close()

			}()

			go func() {

				for {

					switch s.cmdStatus {
					case 1:

						input, err := r.ReadlineEx()
						if err != nil {
							if err != readline.ErrInterrupt {
								res <- err
								return
							}
							if s.cmdStatus == 1 {

								n.Write(common.CMD_SHELL_DATA, s.id, []byte{03})
							}
						}
						if s.cmdStatus == 1 {

							n.Write(common.CMD_SHELL_DATA, s.id, []byte(input+"\n"))
						}

					case 0:
						time.Sleep(time.Millisecond * 100)
					case -1:
						return
					}

				}
			}()
			tick := time.NewTicker(common.CMD_TIMEOUT / 2)
			for {

				select {
				case b := <-s.inChan:
					s.pong = time.Now().Unix()
					if len(b) > 0 {
						b, err := s.translate(b)
						if err != nil {
							c.Println("shell 运行失败", err)
							return
						}

						fmt.Print(string(b))
					}

				case v := <-res:
					if err, ok := v.(error); ok {
						if err.Error() != "退出shell" {
							c.Println("运行shell", param, "失败", err)
						}

					} else {
						c.Println("无法处理消息", v)
					}
					return
				case <-tick.C:
					s.ping = time.Now().Unix()
					if s.ping-s.pong > int64(common.CMD_TIMEOUT/time.Second) {
						c.Println("shell time out")
						return
					}
					n.Write(common.CMD_SHELL_DATA, s.id, nil)
				}
			}
		},
	})
	rootCli.AddCmd(&ishell.Cmd{
		Name: "remoteshell",
		Help: "远程shell",
		Func: func(c *ishell.Context) {
			remoteShell.Run()
		},
	})

}

// 打印节点
func printNodes(c *ishell.Context) {
	l := clientLock.RLock()
	defer l.RUnlock()
	var list []*node

	nodeMap.Range(func(key, value interface{}) bool {
		n := value.(*node)
		list = append(list, n)
		return true
	})
	orderNode(list)
	c.Println("ID  UUID                                  HostName                GOOS          IP                       listenIP")
	c.Println("-----------------------------------------------------------------------------------------------------------------------------")
	for k, n := range list {
		n.id = k + 1
		hostname := bytes.Repeat([]byte(" "), 22)
		copy(hostname, n.hostName)
		ip := bytes.Repeat([]byte(" "), 23)
		if n.uuid == currentNode.uuid {

			copy(ip, "(localhost)"+":"+strconv.Itoa(n.port))
		} else {
			copy(ip, n.addr+":"+strconv.Itoa(n.port))
		}

		listenip := n.mainIp
		goos := bytes.Repeat([]byte(" "), 11)
		copy(goos, n.goos)
		c.Printf("%2d  %s  %s  %s  %s  %s\n", n.id, n.uuid, hostname, goos, ip, listenip)
	}
}

func getRealPath(path string) string {

	path_s := strings.Split(path, "/")
	realpath := []string{}
	if len(path_s) == 0 {
		return "error"
	}
	for _, value := range path_s {

		if value == ".." {
			k := len(realpath)
			kk := k - 1
			realpath = append(realpath[:kk], realpath[k:]...)
		} else {
			realpath = append(realpath, value)
		}
	}

	return strings.Join(realpath, "/")
}
func printConn() {
	connMap.Range(func(key, value interface{}) bool {
		fmt.Println(key)
		return true
	})
}
