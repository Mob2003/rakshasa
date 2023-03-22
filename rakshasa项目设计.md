## 1. rakshasa简介

rakshasa是一个用Go编写的程序，旨在创建一个能够实现**多级代理**，**内网穿透**网络请求。它可以在节点群中任意两个节点之间转发tcp请求和响应，同时支持**socks5代理**，**http代理**，并可**引入外部http、socks5代理池，自动切换请求ip**。

节点之间使用内置证书的TLS加密TCP通信，再叠加一层自定义秘钥的AES加密。该程序可在所有Go支持的平台上使用，包括Windows和Linux服务器。





## 2. 功能要求

- 自动实现多层代理
- 支持Tcp转发代理。
- 支持Tcp反向代理 
- 支持Socks5代理        （包含UDP和TCP6）
- 支持Socks5反向代理
- 支持HTTP代理           （爬虫利器，流量出口支持代理池）
- 去中心化。
- 支持多个节点连接。
- 支持配置文件，可以配置代理服务器的端口、目标服务器的地址和端口、证书文件等信息。
- 支持日志记录，记录代理服务器的请求和响应信息。 
- CLI模式实现远程Shell。
- 执行shellcode。
  
  

## 3. 技术选型

- Go语言：因为Go语言是一门高效的并发编程语言，非常适合网络编程。
- github.com/abiosoft/ishell包：实现各操作系统中的CLI。
- github.com/creack/pty包：实现Linux伪终端，可以执行交互式命令与更高级的终端显示效果。
- github.com/dlclark/regexp2包：全功能正则表达式 。

## 4. 项目结构

项目应按按照以下结构组织：

```go

 ├── aes       //使用标准库crypto实现aes加密
 ├── cert      //证书存放目录，使用embed内嵌到二进制文件
 ├── common    //协议编码格式与配置文件
 ├── gencert   //go实现的证书生成，可以生成临时证书
 ├── httppool  //http代理池检测相关
 ├── readline  //二开以实现更强大的交互式CLI
 ├── readme    //使用文档
 ├── server    //核心代码
        ├──cli.go                       //Cli初始化
        ├──config.go                    //保存读取config.yaml
        ├──conn.go                      //定义结构体，网络连接conn启动，监听，转发
        ├──http_proxy.go                //http代理
        ├──lock.go                      //对标准库的sync.RWMutex进行二次封装，以便于开发调试时找到死锁
        ├──node.go                      //节点代码，实现节点消息广播注册发现
        ├──oder.go                      //排序代码
        ├──raw_tcp_client.go            //tcp转发的client代码
        ├──raw_tcp_server.go            //tcp转发的server代码
        ├──remote_socks5.go             //socks5反向代理
        ├──shell.go                     //交互式CLI
        ├──shell_linux.go               //linux系统下启动shell
        ├──shell_windows.go             //windows系统下启动shell
        ├──shellcode.go                 //执行shellcode
        ├──shellcode_linux.go           //暂未实现
        ├──shellcode_windows.go         //windows下执行shellcode
        ├──socks5.go                    //socks5正向代理
├── main.go
├── config.yaml
├── go.mod
└── go.sum
```

## 5. YML配置详解

程序启动时需指定-f参数来读取YML文件：

```yaml
dstnode:
    - 192.168.1.180:8883 #可以留空，上级节点的ip端口，rakshasa没有公共节点也不会自动发现节点，需要config指定或者启动后使用命令连接其他节点
password: ""                  #通讯秘钥，可以额外指定秘钥，各节点除了证书需要匹配之外，秘钥也需要相同，避免二进制泄漏后被别人无脑连接
port: 8883                     #监听端口
listenip:                         #外网ip，当某个节点掉线后，会尝试连接这个ip
    - 192.168.1.151
limit: false                      #节点掉线后的行为模式，为ture的时候，只连接dstnode指定的ip，不会连接其他节点；为false的时候，尝试连接所有已记录节点的listenip与port
filename: config.yaml    #yaml的文件名，执行保存config命令的时候，会将配置写入这个文件
```

注：节点掉线之后能自动连接，其关键诀窍就是listenip填写公网ip，节点掉线后就会尝试连接已保存的ip和端口。

## 6. 启动参数

- -p                 port
  
        设置本地节点监听端口,默认8883

- -d                用户名:密码@ip:端口 
  
         连接到某个上级节点，多个节点以,隔开，从左到右依次往上连接，例如-d A,B,C，先连接A然后以A为跳板尝试连接B，再以B为跳板尝试连接C

- -connect    ip:port,remote_ip:remote_port
  
       tcp正向代理，本地监听ip:port,将数据通过-d服务器 发送到remote_ip:remote_port

- -bind         ip:port,remote_ip:remote_port
  
       tcp反向代理转发模式， -d服务器 监听remote_ip:remote_port，将数据通过本机节点发送到  ip:port

- -socks5     用户名:密码@ip:端口
  
       以本地socks5代理服务端模式运行，通过-d的服务器多级代理转出数据，如果没有-d参数，则相当于建立了一个本地socks5代理服务器

- -remotesocks5 用户名:密码@ip:端口
  
       在-d的节点上启动一个远程socks5代理，并通过本机节点转出数据

- -http          用户名:密码@ip:端口
  
        以本地http代理服务端模运行，通过-d的服务器多级代理转出数据，如果没有-d参数，则相当于建立了一个本地socks5代理服务器

- -http_proxy_pool  文件路径
  
        http代理最后一层网络出口代理池，参数为文件路径，每个代理一行，每次请求换一个代理

- -check_proxy 单个代理地址或者文件路径
  
        检查http代理是否有效，传入参数可以是ip:port 或者文件,当前支持ipv4,并将结果保存到-check_proxy_out，可选参数-check_proxy_timeout,-check_proxy_url

- -check_proxy_out 文件路径
  
        将有效的代理地址保存到该文件，默认为out.txt

- -check_proxy_timeout int（秒）
  
        检查代理的超时参数

- -check_proxy_url 网址
  
        针对指定的url作为代理池的检测目标，目标返回200ok测为检测通过，默认为https://myip.fireflysoft.net/

- -check_proxy_anonymous true|false
  
        检测代理是否匿名，非匿名代理不保存，默认为true，必须使用默认url

- -f                 文件路径
  
        配置文件路径,为空的时候不读取

- -ip               ip地址
  
        设置本地节点指定公网ip,多个ip以,间隔

- -limit          true|false
  
        limit模式，只连接-d的服务器，不进行额外节点连接,默认为false

- -noshell
  
        不启动shell
  
  

- -password   秘钥
  
        通讯二次加密秘钥，可为空

- -sParam                 string
  
       shellcode的运行参数

- -sTimeout              int（秒）
  
       shellcode的超时等待时间,默认3秒 (default 3)

- -sXor                       string
  
       shellcode的xor解码密钥

- -shellcode                string
  
       与-d配合指定节点执行shellcode,-d参数为空则为本节点执行，可以为base64或者hex编码

## 7. 带参数启动使用例子

```shell
rakshasa -socks5 1080 -d 192.168.1.2:1080 
// 在本地1080端口启动一个socks5，连接到192.168.1.2的节点上，流量出口位于 192,.168.1.2
```

```shell
rakshasa -remotesocks5 1081  -d 192.168.1.2:1080,192.168.1.3:1080 
//方向从右往左(加上本机应该是3个节点)，在192.168.1.3这台机器上开启一个socks5端口1081，流量穿透到本地节点出去
```

```shell
//本地监听并转发到指定ip端口，使用场景本机cs连接teamserver，隐藏本机ip
rakshasa -connect 127.0.0.1:50050,192,168,1,2:50050 -d 192.168.1.3:1080,192.168.1.4:1080
//本机cs连接127.0.0.1:50050实际上通过1.3,1.4节点后，再连接到192.168.1.2:50050 teamserver，teamserver看到你的ip是最后一个节点的ip
```

```shell
//远端监听端口，流量转到本地再出去，为-connect的反向代理模式
rakshasa -bind 192.168.1.2:50050,0,0,0.0:50050 -d 192.168.1.3:1080,192.168.1.4:1080
//与上面相反，在最右端节点监听端口50050，流量到本机节点后，最终发往192.168.1.2，最终上线 IP 为本机 IP
```

## 8. 命令行CLI模式

##### 名词介绍与使用要点

- 每个节点都会生成一个UUID，重启后UUID会改变
- 大部分子命令菜单下，都集成了 new（连接节点）、print（打印节点）、ping（测试节点延迟）功能
- 可用 print 命令，查看在当前节点下，各个节点的 ID、UUID 和 IP 端口。
- 与其他节点进行交互的时候，可以使用ID、UUID、IP:端口去操作，如：
  
```shell
rakshasa>print
ID  UUID                                  HostName                GOOS          IP                       listenIP
-----------------------------------------------------------------------------------------------------------------------------
 1  33fe3ab6-36fe-4636-b684-6f9d9e417981  DESKTOP-KP6GDI2         windows x64  192.168.1.151:8883
rakshasa>ping 1
rakshasa>ping 33fe3ab6-36fe-4636-b684-6f9d9e417981
rakshasa>ping 192.168.1.151:8883
```
- noCLI 不能使用命令行CLI模式

启动rakshasa，默认监听8883，执行help

```shell
start on port: 8883
rakshasa>help

Commands:
  bind              进入bind功能
  clear             clear the screen
  config            配置管理
  connect           进入connect功能
  exit              exit the program
  help              display help
  httpProxy         进入httpProxy功能
  new               与一个或者多个节点连接，使用方法 new ip:端口 多个地址以,间隔 如1080 127.0.0.1:1081,127.0.0.1:1082
  ping              ping 节点
  print             列出所有节点
  remoteShell       远程shell
  remoteSocks5      进入remoteSocks5功能
  shellcode         执行shellcode
  socks5            进入socks5功能
```



连接到其他节点,print当前网络所有节点

```shell
rakshasa>new 127.0.0.1:8881
rakshasa>print
ID  UUID                                  HostName                GOOS          IP                       listenIP
-----------------------------------------------------------------------------------------------------------------------------
 1  33fe3ab6-36fe-4636-b684-6f9d9e417981  DESKTOP-KP6GDI2         windows x64  192.168.1.151:8883
 2  6a6dafb7-ddc9-4da3-afd2-5423038b20ab  DESKTOP-KP6GDI2         windows x64  192.168.1.151:8882
 3  abc6c7aa-52d0-4a33-8688-cd2df23641cd  DESKTOP-KP6GDI2         windows x64  (localhost):8883
 4  e06abfdc-320e-4eb1-9545-8fdf1536fd86  DESKTOP-KP6GDI2         windows x64  192.168.1.151:8880
 5  e19e6d3e-4cdf-4573-9357-d31ee33587f7  DESKTOP-KP6GDI2         windows x64  192.168.1.151:8881
``` 

 测试某个节点延迟

```shell
rakshasa>ping 6a6dafb7-ddc9-4da3-afd2-5423038b20ab
ping 6a6dafb7-ddc9-4da3-afd2-5423038b20ab 674µs
```

特别说明，如果使用UUID操作某一命令，则会自动根据网络连接情况，将消息发送到该节点
如下图例子，我们处于内网的节点A，通过ip可以连接到B公网的机器，不能连到CD，但是B和C是连通，C和D是连通的。
当我们请求D机器的时候，消息会通过B,C最终到达D，不需要特别指定。
![image](https://user-images.githubusercontent.com/128351726/226796357-26276f0f-0a43-4fa1-990c-86b6fadc2e76.png)


#### CLI模式远程shell示例

执行remoteShell进入远程shell

```shell
rakshasa\remoteShell>help

Commands:
  clear      clear the screen
  exit       exit the program
  file       连到节点进行文件管理，参数为id或者uuid
  help       display help
  new        与一个或者多个节点连接，使用方法 new ip:端口 多个地址以,间隔 如1080 127.0.0.1:1081,127.0.0.1:1082
  print      列出所有节点
  shell      反弹shell   使用方法 shell id/uuid 启动参数 ，启动参数可为空，win默认启动cmd，linux默认启动bash， 如 shell 1 powershell 。 shell 1 zsh
```

连接到节点

```shell
rakshasa\remoteShell>shell feed089e-20aa-44d5-bbff-07c968a94ffc
root@virtual-machine:/data/socks5/rakshasa#ls
aes   common  config.yaml  go.mod  main.go   readline  README.md  shell
cert  config  gencert      go.sum  rakshasa.exe  readme    server
root@virtual-machine:/data/socks5/rakshasa# whoami
root
root@virtual-machine:/data/socks5/rakshasa# uname -a
Linux virtual-machine 5.15.0-58-generic #64-Ubuntu SMP Thu Jan 5 11:43:13 UTC 2023 x86_64 x86_64 x86_64 GNU/Linux
root@virtual-machine:/data/socks5/rakshasa#
```

windows下也能执行linux的交互式命令，并正常显示，如htop
![image](https://user-images.githubusercontent.com/128351726/226796526-e2990095-8bf5-4d39-8ef4-1aadc77d684e.png)


注：windows建议使用powershell，cmd下会有一些输入显示bug

#### CLI模式，反向代理bind示例————在远程节点上监听一个端口，转发到本地某个端口

使用方法 new-bind 本地ip:本地port,远程监听ip:远程监听port 目标节点  

```shell
rakshasa\bind>new-bind 127.0.0.1:81,0.0.0.0:8808 192.168.1.180:8883
bind 启动成功
rakshasa\bind>
//在远程机器（192.168.1.180:8883）上，监听0.0.0.0:8808，将所有数据，通过本地节点，发送到127.0.0.1:81,(当然也可以是内网其他电脑ip端口或者其他公网)
rakshasa\bind>list
当前连接数量: 1
ID 1 本地端口 127.0.0.1:81 远程端口 0.0.0.0:8808 服务器uuid feed089e-20aa-44d5-bbff-07c968a94ffc
//feed089e-20aa-44d5-bbff-07c968a94ffc是（192.168.1.180:8883）的uuid
```





#### http代理池使用示例

```shell
D:\rakshasa>rakshasa -check_proxy in.txt
2023/03/19 19:24:55 地址 104.223.135.178:10000 通过匿名代理检测
2023/03/19 19:24:58 地址 169.55.89.6:80 通过匿名代理检测
2023/03/19 19:25:52 一共有2个代理通过检测，已保存到 out.txt

D:\rakshasa>rakshasa -http_proxy_pool out.txt -http_proxy 8080
start on port: 8883
httpProxy start  :8080
2023/03/19 19:23:26 httpProxy 启动成功
rakshasa>
```

使用curl查一下ip，请求所使用的代理每次请求都会改变

```shell
C:\Users\Administrator>curl -x http://127.0.0.1:8080 https://myip.fireflysoft.net/
169.55.89.6
C:\Users\Administrator>curl -x http://127.0.0.1:8080 https://myip.fireflysoft.net/
104.223.135.178
```
