package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"rakshasa/aes"
	"rakshasa/cert"
	"rakshasa/common"
	"rakshasa/httppool"
	"rakshasa/server"
	"strconv"
	"strings"
	"sync"
)

func main() {

	var (
		//以下为配置参数
		dstNode               = flag.String("d", "", "依次连接到指定的 上级节点地址，格式为 ip:端口 多个节点以,隔开\r\n	-d 192.168.1.1:8883\r\n	-d 192.168.1.1:8883,192.168.1.2:8882")
		limit                 = flag.String("limit", "", "limit模式，只连接-d的节点，不进行额外节点连接,默认为false，如果为true，本节点掉线的时候，将会尝试连接所有已保存节点")
		password              = flag.String("password", "", "通讯二次加密秘钥，可为空")
		listenip              = flag.String("ip", "", "设置本地节点指定公网ip,多个ip以,间隔，如\r\n	-ip 192.168.1.1")
		port                  = flag.String("p", "", "设置本地节点监听端口,默认8883")
		configFile            = flag.String("f", "", "配置文件路径,为空的时候不读取")
		check_proxy           = flag.String("check_proxy", "", "检查http代理是否有效，传入参数可以是ip:port 或者文件,当前支持ipv4,并将结果保存到-check_proxy_out，可选参数-check_proxy_timeout,-check_proxy_url,使用方法: \r\n	-check_proxy 192.168.1.1:8080\r\n	-check_proxy in.txt\r\n	-check_proxy in.txt -check_proxy_out out.txt -check_proxy_timeout 10 -check_proxy_url https://www.google.com/")
		check_proxy_out       = flag.String("check_proxy_out", "out.txt", "配合-check_proxy一起，将有效代理保存到指定文件,默认保存到 out.txt")
		check_proxy_timeout   = flag.Uint("check_proxy_timeout", 10, "配合-check_proxy一起，设定代理检测超时，单位秒，默认10")
		check_proxy_url       = flag.String("check_proxy_url", server.CheckProxyUrl, "自定义url检测，配合-check_proxy一起使用，设定代理检测url,不能测试是否匿名")
		check_proxy_anonymous = flag.Bool("check_proxy_anonymous", true, "检测代理是否匿名，非匿名代理不保存，默认为true，必须使用默认url")

		//以下为功能参数,必须配合-d参数启动
		socks5port       = flag.String("socks5", "", "以本地socks5代理服务端模式运行，通过-d的服务器多级代理转出数据，如果没有-d参数，则相当于建立了一个本地socks5代理服务器,如: -socks5 admin:12345@0.0.0.0:1080")
		remoteSocksport  = flag.String("remotesocks5", "", "-d节点监听socks5代理，并将请求通过本地转出,如: -remote admin:12345@0.0.0.0:1080")
		rawbind          = flag.String("bind", "", "反向代理转发模式,格式为ip:port,remote_ip:remote_port，-d指定节点将会监听remote_ip:remote_port,通过本机将数据转发到ip:port,如\r\n	 -bind 127.0.0.1:80,0.0.0.0:80")
		rawconnect       = flag.String("connect", "", "代理转发模式,格式为ip:port,remote_ip:remote_port，本地监听ip:port,并在-d节点连接到remote_ip:remote_port,如\r\n   	-connect 0.0.0.0:80,192.168.1.1:80")
		noCLI            = flag.Bool("nocli", false, "不启动cli")
		shellCode        = flag.String("shellcode", "", "与-d配合指定节点执行shellcode,-d参数为空则为本节点执行，可以为base64或者hex编码")
		shellCodeXorKey  = flag.String("sXor", "", "shellcode的xor解码密钥")
		shellCodeParam   = flag.String("sParam", "", "shellcode的运行参数")
		shellCodeTimeout = flag.Int("sTimeout", 3, "shellcode的超时等待时间,默认3秒")
		http_proxy       = flag.String("http_proxy", "", "以本地http代理服务端模式运行，通过-d的服务器多级代理转出数据，如果没有-d参数，则使用本机进行下一步连接， 用户名:密码@ip:端口 可以省略为端口,如: \r\n	-http_proxy admin:12345@0.0.0.0:8080\r\n	-http_proxy admin:12345@8080\r\n	-http_proxy 8080")
		http_proxy_pool  = flag.String("http_proxy_pool", "", "从指定文件读取http代理服务器池，通过最后节点后（不使用-d则为本机），再从该池里读取一个代理进行请求")
	)

	flag.Parse()
	if *check_proxy != "" {
		if *check_proxy_url != server.CheckProxyUrl && *check_proxy_anonymous == true {
			log.Println("检测url不是默认url，将取消匿名代理检测")
			*check_proxy_anonymous = false
		}
		httppool.CheckProxy(*check_proxy, *check_proxy_out, *check_proxy_timeout, *check_proxy_url, *check_proxy_anonymous)
		return
	}

	var config common.Config
	if *configFile != "" {
		if err := server.ConfigLoad(*configFile); err != nil {
			log.Fatalln("读取配置文件", *configFile, "失败 ", err)
		}
		config = server.GetConfig()
	} else {
		config = common.Config{
			Port:     8883,
			Limit:    false,
			FileName: "config.yaml",
		}

	}

	if *dstNode != "" {
		serverlist, err := common.ResolveTCPAddr(*dstNode)
		if err != nil {
			log.Fatalln("-d参数错误", err)
		}
		config.DstNode = serverlist
	}

	if *password != "" {
		config.Password = *password
	}
	if *listenip != "" {
		config.ListenIp = strings.Split(*listenip, ",")
	}
	if *limit != "" {
		if *limit != "flase" && *limit != "true" {
			log.Fatalln("limit 参数错误，必须是 false 或者 true")
		}
		config.Limit = *limit == "true"
	}
	if *port != "" {
		p, _ := strconv.Atoi(*port)
		if p < 1 || p > 65535 {
			log.Fatalln("port 参数错误，必须是1-65535")
		}
		config.Port = p
	}
	server.SetConfig(config)
	//修正dstNode为空字串的bug
	for i := len(config.DstNode) - 1; i >= 0; i-- {
		addr := config.DstNode[i]
		if addr == "" {
			config.DstNode = append(config.DstNode[:i], config.DstNode[i+1:]...)
		}
	}
	server.SetConfig(config)

	//设置一下秘钥
	aes.Key = aes.MD5_B(config.Password + string(cert.PublicKey[:16]))
	//初始化node
	server.InitCurrentNode()

	if common.Debug {
		go func() {
			err := http.ListenAndServe("0.0.0.0:8083", nil)
			if err != nil {
				err = http.ListenAndServe("0.0.0.0:8084", nil)
				if err != nil {
					err = http.ListenAndServe("0.0.0.0:8085", nil)
				}
			}
		}()
	}
	//启动节点
	if len(config.DstNode) > 0 && config.DstNode[0] != "" {
		if _, err := server.GetNodeFromAddrs(config.DstNode); err != nil {
			log.Fatalln("连接节点失败", err)
		}
	}

	if *shellCode != "" {

		server.RunShellcodeWithDst(*dstNode, *shellCode, *shellCodeXorKey, *shellCodeParam, *shellCodeTimeout)

	}
	if err := server.StartServer(config.Port); err != nil {
		log.Fatalln(err)
	}

	//如果有参数启动，启动一下
	if *rawbind != "" {
		if *dstNode == "" {
			log.Fatalln("请以 -d 输入远程服务器ip地址")
		}

		if err := server.StartRawBind(*rawbind, config.DstNode); err != nil {
			log.Fatalln("bind启动失败", err)
		}
		log.Println("rawBind启动成功")
	} else if *rawconnect != "" {
		if *dstNode == "" {
			log.Fatalln("请以 -d 输入远程服务器ip地址")
		}
		n, err := server.GetNodeFromAddrs(config.DstNode)
		if err != nil {
			log.Fatalln("connect启动失败", err)
		}
		if err := server.StartRawConnect(*rawconnect, n); err != nil {
			log.Fatalln("connect启动失败", err)
		}
		log.Println("rawConnect启动成功")
	} else if *socks5port != "" {

		cfg, err := common.ParseAddr(*socks5port)
		if err != nil {
			log.Fatalln(err)
		}
		if err := server.StartSocks5(cfg, config.DstNode); err != nil {
			log.Fatalln("socks5启动失败", err)
		}
		log.Println("socks5启动成功")

	} else if *remoteSocksport != "" {
		if *dstNode == "" {
			log.Fatalln("请以 -d 输入远程服务器ip地址")
		}
		n, err := server.GetNodeFromAddrs(config.DstNode)
		if err != nil {
			log.Fatalln("remoteSocks5启动失败", err)
		}
		cfg, err := common.ParseAddr(*remoteSocksport)
		if err != nil {
			log.Fatalln(err)
		}
		if err := server.StartRemoteSocks5(cfg, n); err != nil {
			log.Fatalln("remoteSocks5启动失败", err)
		}
		log.Println("remoteSocks5 启动成功")
	} else if *http_proxy != "" {
		cfg, err := common.ParseAddr(*http_proxy)
		if err != nil {
			log.Fatalln(err)
		}
		if err := server.StartHttpProxy(cfg, config.DstNode, *http_proxy_pool); err != nil {
			log.Fatalln("httpProxy启动失败", err)
		}
		log.Println("httpProxy 启动成功")
	}

	if !*noCLI {
		common.SetConsoleVT()
		server.CliRun()

	} else {
		var wait sync.WaitGroup
		wait.Add(1)
		wait.Wait()
	}

}
