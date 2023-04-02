package httppool

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"rakshasa_lite/common"
	"strings"
	"sync"
)

type HttpPool struct {
	r *bufio.Reader
	f *os.File
	sync.Mutex
}

func HttpPoolInit(file string) (*HttpPool, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("打开http代理池文件 %s 失败", file)
	}
	p := &HttpPool{
		r:     bufio.NewReader(f),
		f:     f,
		Mutex: sync.Mutex{},
	}
	if _, err = p.do_next(0); err != nil {
		return nil, fmt.Errorf("无法从%s文件获取代理，错误%v", file, err)
	}
	return p, nil
}
func (p *HttpPool) Next() *common.Addr {
	addr, _ := p.do_next(0)
	return addr
}
func (p *HttpPool) do_next(n int) (*common.Addr, error) {
	if n > 100 {
		return nil, errors.New("重试错误次数过多")
	}
	p.Lock()
	line, err := p.r.ReadString(10)
	if err == io.EOF {
		p.f.Seek(0, 0)
		p.r.Reset(p.f)
		p.Unlock()
		return p.do_next(n + 1)
	}
	p.Unlock()
	line = strings.TrimRight(line, "\n")
	line = strings.TrimRight(line, "\r")

	if len(line) == 0 {
		return p.do_next(n + 1)
	}
	addr, err := common.ParseAddr(line)
	if err != nil {
		return p.do_next(n + 1)
	}
	return addr, nil
}
