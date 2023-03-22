package common

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/dlclark/regexp2"
	"net"
	"strconv"
)

type Addr struct {
	scheam                  string
	user, passwd            string
	ip                      string
	port                    int
	httpAuthorizationHeader string
}

func ParseAddr(str string) (cfg *Addr, err error) {
	defer func() {
		if cfg != nil && cfg.user != "" && cfg.passwd != "" {
			cfg.httpAuthorizationHeader = fmt.Sprintf("Proxy-Authorization: Basic %s", base64.URLEncoding.EncodeToString([]byte(cfg.user+":"+cfg.passwd)))

		}

	}()
	r, _ := regexp2.Compile(`^(http://|socks5://)?(\S+):(\S+)@(\S+):(\d+)`, 0)
	m, err := r.FindStringMatch(str)
	if err != nil {
		return nil, errors.New("配置解析错误，" + err.Error())
	}
	if m != nil {
		var res_v []string
		for _, v := range m.Groups() {
			res_v = append(res_v, v.String())
		}
		addr, err := net.ResolveTCPAddr("tcp", res_v[4]+":"+res_v[5])
		if err != nil {
			return nil, errors.New("配置解析错误 " + res_v[4] + ":" + res_v[5] + " 不是有效的 地址:端口")
		}
		return &Addr{
			scheam: res_v[1],
			user:   res_v[2],
			passwd: res_v[3],
			ip:     res_v[4],
			port:   addr.Port,
		}, nil
	}
	r, _ = regexp2.Compile(`^(http://|socks5://)?(\S+):(\S+)@(\d+)`, 0)
	m, err = r.FindStringMatch(str)
	if err != nil {
		return nil, errors.New("配置解析错误，" + err.Error())
	}
	if m != nil {
		var res_v []string
		for _, v := range m.Groups() {
			res_v = append(res_v, v.String())
		}
		port, _ := strconv.Atoi(res_v[4])

		return &Addr{
			scheam: res_v[1],
			user:   res_v[2],
			passwd: res_v[3],
			ip:     "",
			port:   port,
		}, nil
	}
	r, _ = regexp2.Compile(`^(http://|socks5://)?(\S+):(\S+)$`, 0)
	m, err = r.FindStringMatch(str)
	if err != nil {
		return nil, errors.New("配置解析错误，" + err.Error())
	}
	if m != nil {
		var res_v []string
		for _, v := range m.Groups() {
			res_v = append(res_v, v.String())
		}
		addr, err := net.ResolveTCPAddr("tcp", res_v[2]+":"+res_v[3])
		if err != nil {
			return nil, errors.New("配置解析错误 " + res_v[1] + ":" + res_v[2] + " 不是有效的 地址:端口")
		}

		return &Addr{
			scheam: res_v[1],
			user:   "",
			passwd: "",
			ip:     res_v[2],
			port:   addr.Port,
		}, nil
	}

	port, err := strconv.Atoi(str)
	if err != nil {
		return nil, errors.New("配置解析错误，请按照 用户名:密码@地址:端口 的方式填写，或者 用户名:密码@端口 或者 ip:端口 或者 只有端口")
	}
	return &Addr{port: port}, nil
}
func (c *Addr) IP() string {
	return c.ip
}
func (c *Addr) Addr() string {
	return fmt.Sprintf("%s:%d", c.ip, c.port)
}
func (c *Addr) Port() string {
	return fmt.Sprintf("%d", c.port)
}
func (c *Addr) String() string {
	if c.user == "" && c.passwd == "" {
		if c.ip == "" {
			return fmt.Sprintf("%d", c.port)
		}

		return fmt.Sprintf("%s%s:%d", c.scheam, c.ip, c.port)
	}

	return fmt.Sprintf("%s%s:%s@%s:%d", c.scheam, c.user, c.passwd, c.ip, c.port)
}
func (c *Addr) GetHttpAuthorizationHeader() string {
	return c.httpAuthorizationHeader
}
func (c *Addr) User() string {
	return c.user
}
func (c *Addr) Password() string {
	return c.passwd
}
func (c *Addr) Scheam() string {
	return c.scheam
}
func (c *Addr) HttpUrl() string {
	return "http://" + c.Addr()
}
