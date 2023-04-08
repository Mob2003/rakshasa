package server

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"rakshasa/common"
	"strings"
	"sync"
	"time"
)

func CheckProxy(in, out string, timeout uint, checkurl string, anonymous bool) {
	outFile, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("CheckProxy 无法写出文件 %s", out)
	}
	defer outFile.Close()
	var res int
	if cfg, err := common.ParseAddr(in); err == nil {
		if check(cfg, timeout, checkurl, outFile, anonymous) {
			res = 1
		}
	} else {
		inFile, err := os.OpenFile(in, os.O_RDONLY, 0666)
		if err != nil {
			log.Fatalf("-check_proxy 无法识别无法读取 %s", in)
		}
		limit := make(chan struct{}, 32)
		reader := bufio.NewReader(inFile)
		var wg sync.WaitGroup
		for {
			line, err := reader.ReadString(10)
			if err == io.EOF && line == "" {
				break
			}
			wg.Add(1)
			limit <- struct{}{}
			go func() {
				line = strings.TrimRight(line, "\n")
				line = strings.TrimRight(line, "\r")

				if cfg, err = common.ParseAddr(line); err == nil {
					if check(cfg, timeout, checkurl, outFile, anonymous) {
						res++
					}
				}

				<-limit
				wg.Done()
			}()
		}
		wg.Wait()
	}
	log.Printf("一共有%d个代理通过检测，已保存到 %v", res, out)

}
func check(cfg *common.Addr, timeout uint, checkurl string, outFile *os.File, anonymous bool) bool {

	switch cfg.Scheam() {
	case "", "http://":

		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(cfg.HttpUrl())
		}

		transport := &http.Transport{Proxy: proxy}

		client := &http.Client{Transport: transport, Timeout: time.Second * time.Duration(timeout)}
		resp, err := client.Get(checkurl)

		if err != nil {

			if strings.Contains(err.Error(), "Client.Timeout") {
				//log.Printf("地址 %v 检测失败 超时没响应\r\n", cfg)
				//} else {
				//log.Printf("地址 %v 检测失败 %v\r\n", cfg, err)
			}

			return false
		}

		if resp.StatusCode == 200 {
			if anonymous {
				b, _ := io.ReadAll(resp.Body)
				if string(b) == cfg.IP() {
					log.Printf("地址 %v 通过匿名代理检测\r\n", cfg.String())
					if _, err = outFile.WriteString(cfg.String() + "\r\n"); err != nil {
						log.Printf("无法写入文件%s", outFile.Name())
					}
					if err = outFile.Sync(); err != nil {
						log.Printf("无法写入文件%s", outFile.Name())
					}
				} else {
					return false
				}
			} else {
				log.Printf("地址 %v 通过检测\r\n", cfg.String())
				if _, err = outFile.WriteString(cfg.String() + "\r\n"); err != nil {
					log.Printf("无法写入文件%s", outFile.Name())
				}
				if err = outFile.Sync(); err != nil {
					log.Printf("无法写入文件%s", outFile.Name())
				}
			}

			return true
		}
		//log.Printf("地址 %v 检测失败 结果状态码不是 200\r\n", addr)
		return false
	case "socks5://":

		netconn, err := net.Dial("tcp", cfg.Addr())

		if err != nil {
			return false
		}
		netconn.Write([]byte{5, 1, 2})
		var result [8192]byte
		n, err := netconn.Read(result[:])
		if err != nil {
			return false
		}

		if string(result[:n]) == string([]byte{5, 2}) { //需要认证
			user, password := cfg.User(), cfg.Password()
			if user == "" && password == "" {
				return false
			}

			data := make([]byte, (3 + len(user) + len(password)))
			data[0] = 5
			data[1] = byte(len(user))
			copy(data[2:], user)
			data[2+len(user)] = byte(len(password))
			copy(data[3+len(user):], password)
			netconn.Write(data)
			n, err = netconn.Read(result[:])
			if err != nil || string(result[:n]) != string([]byte{5, 0}) {
				return false
			}
		}
		log.Printf("地址 %v 通过socks5检测\r\n", cfg.String())
		if _, err = outFile.WriteString(cfg.String() + "\r\n"); err != nil {
			log.Printf("无法写入文件%s", outFile.Name())
		}
		if err = outFile.Sync(); err != nil {
			log.Printf("无法写入文件%s", outFile.Name())
		}
		return true
	default:

	}
	return false
}


