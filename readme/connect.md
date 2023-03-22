[主菜单](./shell.md)

### new-connect

从本机监听一个tcp端口，将请求转发到 目标节点连接到指定ip:port，正向tcp代理

使用方法 new-connect ip:port,remote_ip:remote_port 目标节点  
如 new-connect 0.0.0.0:88,192.168.1.180:8808  44b5b521-719c-4e2b-b069-ad176d8d88ba

注：连接方向从左往右，如上例子，从本节点监听0.0.0.0:88，并将连接请求通过目标节点发送到192.168.1.180:8808

### list

打印出当前connect连接情况

### close

关掉一个connect

使用方法,先使用list获得当前执行的connect的id，再执行 close id



```shell
rakshasa>connect
rakshasa\connect>new-connect 0.0.0.0:88,192.168.1.180:8808 44b5b521-719c-4e2b-b069-ad176d8d88ba
connect连接 44b5b521-719c-4e2b-b069-ad176d8d88ba 成功
rakshasa\connect>list
当前连接数量: 1
ID 2 本地端口 0.0.0.0:88 远程端口 192.168.1.180:8808 服务器uuid 44b5b521-719c-4e2b-b069-ad176d8d88ba
rakshasa\connect>close 2
```