//go:build windows
// +build windows

package server

import (
	"errors"
	"io"
	"rakshasa_lite/common"
	"time"

	"os/exec"
)

func startCMD(n *node, msgid uint32, param StartCmdParam) error {
	if param.Param == "" {
		param.Param = "cmd"
	}
	shellMapLock.Lock()
	defer func() {

		shellMapLock.Unlock()
	}()

	cmd := &remoteCmd{
		id:        common.GetID(),
		inChan:    make(chan []byte),
		translate: func(in []byte) ([]byte, error) { return in, nil },
		pong:      time.Now().Unix(),
	}

	c := exec.Command("chcp")
	res, err := c.Output()
	if err != nil {

		return err
	}

	cmd.cmd = exec.Command(param.Param)

	stdout, err := cmd.cmd.StdoutPipe()
	if err != nil {

		return err
	}
	cmd.stdin, err = cmd.cmd.StdinPipe()
	if err != nil {

		return err
	}

	stderr, err := cmd.cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.cmd.Start()
	if err != nil {
		return err
	}
	outErr := make(chan error, 999)
	n.shellMap.Store(cmd.id, cmd)

	go func(cmd *remoteCmd) {
		defer func() {
			n.shellMap.Delete(cmd.id)
			stdout.Close()
			cmd.stdin.Close()
			cmd.cmd.Process.Kill()
		}()
		var errchan = make(chan error, 10)
		go func() {
			for {
				select {
				case b := <-cmd.inChan:
					if len(b) == 0 { //ping数据包
						n.Write(common.CMD_SHELL_DATA, cmd.id, nil) //pong
					} else {
						_, err = cmd.stdin.Write(b)
						if err != nil {
							errchan <- err
						}
					}

				case err = <-errchan:

					cmd.cmd.Process.Kill()
				case err = <-outErr:
					n.Write(common.CMD_SHELL_RESULT, msgid, append([]byte{0}, err.Error()...))
					cmd.cmd.Process.Kill()
					return
				case <-time.After(common.CMD_TIMEOUT): //避免超时

					cmd.cmd.Process.Kill()
					return

				}
			}
		}()
		go func() {

			buf := make([]byte, common.MAX_PLAINTEXT)
			for {
				num, err2 := stdout.Read(buf)
				if err2 != nil || io.EOF == err2 {
					outErr <- errors.New("退出shell")

					break
				}

				n.Write(common.CMD_SHELL_DATA, cmd.id, buf[:num])

			}

		}()
		go func() {
			buf := make([]byte, 1024)
			for {
				num, err2 := stderr.Read(buf)
				if err2 != nil || io.EOF == err2 {

					break
				}
				n.Write(common.CMD_SHELL_DATA, cmd.id, buf[:num])
				//output, _ := libraries.GbkToUtf8(buf[:n])

			}
		}()
		cmd.cmd.Wait()
	}(cmd)

	n.Write(common.CMD_SHELL_RESULT, msgid, append([]byte{1, byte(cmd.id), byte(cmd.id >> 8), byte(cmd.id >> 16), byte(cmd.id >> 24), 0}, res...))
	return nil
}
