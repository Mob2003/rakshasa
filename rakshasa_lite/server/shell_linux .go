//go:build linux || darwin
// +build linux darwin

package server

import (
	"errors"
	"io"
	"os/exec"
	"rakshasa_lite/common"
	"time"

	"github.com/creack/pty"
)

func startCMD(n *node, msgid uint32, param StartCmdParam) error {
	if param.Param == "" {
		param.Param = "/bin/bash"
	}
	shellMapLock.Lock()
	defer func() {
		shellMapLock.Unlock()
	}()

	cmd := &remoteCmd{
		id:     common.GetID(),
		inChan: make(chan []byte),

		translate: func(in []byte) ([]byte, error) { return in, nil },
		pong:      time.Now().Unix(),
	}

	cmd.cmd = exec.Command(param.Param)
	f, err := pty.StartWithSize(cmd.cmd, param.Size)
	if err != nil {

		return err
	}
	cmd.stdin = f
	outErr := make(chan error, 999)

	n.shellMap.Store(cmd.id, cmd)

	go func(cmd *remoteCmd) {
		defer func() {
			n.shellMap.Delete(cmd.id)
			f.Close()
			cmd.stdin.Close()
		}()
		errChan := make(chan error, 999)
		go func() {
			for {
				select {
				case b := <-cmd.inChan:
					if len(b) == 0 { //ping数据包

						n.Write(common.CMD_SHELL_DATA, cmd.id, nil) //pong
					} else {

						_, err = cmd.stdin.Write(b)
						if err != nil {
							errChan <- err
						}
					}

				case err = <-errChan:

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
				num, err2 := f.Read(buf)
				if err2 != nil || io.EOF == err2 {
					outErr <- errors.New("退出shell")
					break
				}

				n.Write(common.CMD_SHELL_DATA, cmd.id, buf[:num])

			}

		}()

		cmd.cmd.Wait()
	}(cmd)
	n.Write(common.CMD_SHELL_RESULT, msgid, []byte{1, byte(cmd.id), byte(cmd.id >> 8), byte(cmd.id >> 16), byte(cmd.id >> 24), 1})
	return nil

}
