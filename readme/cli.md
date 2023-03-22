## 目录



- [connect](./connect.md): 端口代理，本节点监听端口，并通过出口节点连接到指定的ip端口.
- [bind](./bind.md): 反向代理，出口节点监听端口，通过本节点连接到ip端口.
- [socks5](./socks5.md): socks5代理，本节点启动socks5代理，通过出口节点连接目标.
- [remotesocks5](./remotesocks5.md): 反向socks5代理，目标节点启动socks5，并通过本节点连接目标.
- [http](./http.md): http代理，本节点启动http代理，通过出口节点连接目标.
- [remoteshell](./remoteshell.md): 远程shell.
- [shellcode](./shellcode.md): windows执行shellcode,linux未实现.
- [config](./config.md): 配置管理.



所有子功能目录下都可以执行下面三个方法
可以通过输入首字母+tab进行自动补全

### new
连接一个新的服务器，参数必须是ip:port

### print
打印出已连接的节点

### ping
尝试对节点发送ping，并输出收到pong一共需要的时间

使用方法,ping id/uuid/ip:port

```shell
rakshasa>new 192.168.1.137:8884
rakshasa>print
ID  UUID                                  HostName                GOOS          IP                       listenIP
-----------------------------------------------------------------------------------------------------------------------------
 1  12a8f492-5fff-4b75-935a-533a276d546e  DESKTOP-DAAI4F1         windows x64  (localhost):8883
 2  44b5b521-719c-4e2b-b069-ad176d8d88ba  DESKTOP-DAAI4F1         windows x64  192.168.1.137:8884
rakshasa>ping 2
ping 44b5b521-719c-4e2b-b069-ad176d8d88ba 0s
rakshasa>ping 44b5b521-719c-4e2b-b069-ad176d8d88ba
ping 44b5b521-719c-4e2b-b069-ad176d8d88ba 0s
rakshasa>ping 192.168.1.137:8884
ping 44b5b521-719c-4e2b-b069-ad176d8d88ba 0s
```