[主菜单](./cli.md)

### new-socks5 
以本地socks5代理服务端模式运行，通过远程节点转发代理连接到指定目标，如果远程节点为空，则为本地直接连

使用方法 new-socks5 port 目标节点  
如 new-socks5 1080  57a3edbc-3120-48b3-95e3-bd712a5e2fe9



### list
打印出当前socks5连接情况

### close
关掉一个socks5

使用方法,先使用list获得当前执行的socks5的id，再执行 close id

```shell
rakshasa>socks5
rakshasa\socks5>print
ID  UUID                                  HostName                GOOS          IP                       listenIP
-----------------------------------------------------------------------------------------------------------------------------
 1  c105b8d2-77c7-462d-b0da-6d9785f77234  DESKTOP-DAAI4F1         windows x64  (localhost):8883
 2  e309f028-84ae-4452-88ab-83f1deab0cf4  DESKTOP-DAAI4F1         windows x64  192.168.1.137:8884
rakshasa\socks5>new-socks5 2
socks5 start  :2
本地socks5启动成功
rakshasa\socks5>list
当前监听端口数量: 1
ID 2 本地监听端口 :2 转发服务器uuid c105b8d2-77c7-462d-b0da-6d9785f77234
rakshasa\socks5>close 2
rakshasa\socks5>
```