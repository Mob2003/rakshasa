package server

import (
	"errors"
	"fmt"
	"rakshasa/common"
	"strconv"
	"strings"
	"time"

	"github.com/luyu6056/ishell"
)

func StartRemoteSocks5(cfg *common.Addr, n *node) error {

	l := &clientListen{
		id:         common.GetID(),
		localAddr:  "",
		remoteAddr: cfg.Addr(),
		server:     n,
		typ:        "socks5",
		result:     make(chan interface{}),
	}
	l.openOption = common.CMD_REMOTE_SOCKS5
	l.openMsg = []byte(cfg.String())
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
func init() {
	remoteSocks5shell := cliInit()
	remoteSocks5shell.SetPrompt("rakshasa\\remotesocks5>")
	remoteSocks5shell.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "列出当前连接的ID和其他信息",
		Func: func(c *ishell.Context) {

			var list []*clientListen
			currentNode.listenMap.Range(func(key, value interface{}) bool {
				if v, ok := value.(*clientListen); ok {
					if v.typ == "socks5" {
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

	remoteSocks5shell.AddCmd(&ishell.Cmd{
		Name: "new-remotesocks5",
		Help: "新建一个remotesocks5连接到本节点，使用方法 new-remotesocks5 配置字串符 目标服务器  如 new-remotesocks5 admin:123456@0.0.0.0:1080 127.0.0.1:1081",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 2 {
				c.Println("参数错误")
				return
			}

			n, err := GetNodeFromAddrs(strings.Split(c.Args[1], ","))
			if err != nil {
				c.Println("无法连接 ", c.Args[1], err)
				return
			}
			cfg, err := common.ParseAddr(c.Args[0])
			if err != nil {
				c.Println(err)
				return
			}
			if err = StartRemoteSocks5(cfg, n); err != nil {
				c.Println("连接", c.Args[1], "失败", err)
				return
			}
			c.Println("节点", c.Args[1], "配置信息,", c.Args[0], ",启动socks5 到 本节点 成功")
		},
	})
	remoteSocks5shell.AddCmd(&ishell.Cmd{
		Name: "close",
		Help: "关闭一个remotesocks5连接，使用方法 close ID",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("参数错误，例子 close 1")
				return
			}
			id, _ := strconv.Atoi(c.Args[0])

			var l *clientListen
			if value, ok := currentNode.listenMap.Load(uint32(id)); ok {
				if v, ok := value.(*clientListen); ok && v.typ == "socks5" {
					l = v
				}

			}
			if l == nil {
				c.Println("没有找到ID为", id, "的连接")
			} else {
				l.Close("命令行关闭")
				l.server.Write(common.CMD_DELETE_LISTEN, l.id, nil)
				currentNode.listenMap.Delete(uint32(id))

			}
		},
	})
	rootCli.AddCmd(&ishell.Cmd{
		Name: "remotesocks5",
		Help: "进入remotesocks5功能",
		Func: func(c *ishell.Context) {

			remoteSocks5shell.Run()

		},
	})
}
