package server

/*
 *高级shell功能
 *node节点管理、remoteShell远程shell，config配置管理
 */
import (
	"github.com/creack/pty"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

var (
	shellMapLock sync.Mutex
)

type StartCmdParam struct {
	Param string
	Size  *pty.Winsize
}
type remoteCmd struct {
	cmdStatus  int32
	cmd        *exec.Cmd
	id         uint32
	stdin      io.WriteCloser
	inChan     chan []byte
	translate  func(in []byte) ([]byte, error)
	ping, pong int64
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

func getNode(arg string) (*node, error) {
	l := clientLock.RLock()

	id, err := strconv.Atoi(arg)

	if err == nil {
		for _, n := range nodeMap {
			if n.id == id && n.uuid != currentNode.uuid {
				l.RUnlock()
				return n, nil
			}
		}
	} else {
		if v, ok := nodeMap[arg]; ok && v.uuid != currentNode.uuid {
			l.RUnlock()
			return v, nil
		}
	}
	l.RUnlock()

	return connectNew(arg)
}
func getNodeWithCurrentNode(arg string) (*node, error) {
	l := clientLock.RLock()

	id, err := strconv.Atoi(arg)

	if err == nil {
		for _, n := range nodeMap {
			if n.id == id {
				l.RUnlock()
				return n, nil
			}
		}
	} else {
		if v, ok := nodeMap[arg]; ok {
			l.RUnlock()
			return v, nil
		}
	}
	l.RUnlock()

	return connectNew(arg)
}
