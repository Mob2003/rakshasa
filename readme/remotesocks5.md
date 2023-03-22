[主菜单](./shell.md)

### new-remotesocks5

远程节点启动socks5代理，通过本节点连接到指定目标

使用方法 new-remotesocks5 port 目标节点  
如 new-remotesocks5 1080  57a3edbc-3120-48b3-95e3-bd712a5e2fe9

注：与socks5相反，是远程目标节点开启socks5，通过本节点转发输出

### list

打印出当前remoteSocks5连接情况

### close

关掉一个remoteSocks5

使用方法,先使用list获得当前执行的remoteSocks5的id，再执行 close id



```shell
rakshasa>remotesocks5
rakshasa\remotesocks5>print
ID  UUID                                  HostName                GOOS          IP                       listenIP
-----------------------------------------------------------------------------------------------------------------------------
 1  c105b8d2-77c7-462d-b0da-6d9785f77234  DESKTOP-DAAI4F1         windows x64  (localhost):8883
 2  e309f028-84ae-4452-88ab-83f1deab0cf4  DESKTOP-DAAI4F1         windows x64  192.168.1.137:8884
rakshasa\remotesocks5>new-remotesocks5 1080 2
节点 2 配置信息, 1080 ,启动socks5 到 本节点 成功
rakshasa\remotesocks5>list
当前连接数量: 1
ID 1 本地端口  远程端口 :1080 服务器uuid e309f028-84ae-4452-88ab-83f1deab0cf4
rakshasa\remotesocks5>close 1
rakshasa\remotesocks5>
```