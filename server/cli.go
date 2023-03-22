package server

import (
	"rakshasa/common"
	"strings"
	"time"

	"github.com/luyu6056/ishell"
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
				_, err := connectNew(addr)
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
				n, ok := nodeMap[c.Args[0]]
				if ok {
					n.Delete("")
				}

			},
		})
	}
	return shell
}
