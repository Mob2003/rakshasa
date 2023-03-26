[主菜单](./cli.md)

### new-bind

从指定节点启动端口监听tcp请求，通过本节点连接到指定ip端口，反向代理

使用方法 new-bind ip:port,remote_ip:remote_port 目标节点  
如 new-bind 192.168.1.180:8808,0.0.0.0:88  44b5b521-719c-4e2b-b069-ad176d8d88ba

注意:与connect相反，从右往左连接，如上面例子，远程节点监听0.0.0.0:88，并将所有连接通过本节点发送到192.168.1.180:8808

### list

打印出当前bind连接情况

### close

关掉一个bind

使用方法,先使用list获得当前执行的bind的id，再执行 close id



```shell
rakshasa>bind
rakshasa\bind>new-bind 192.168.1.180:8808,0.0.0.0:88 44b5b521-719c-4e2b-b069-ad176d8d88ba
bind 启动成功
rakshasa\bind>list
当前连接数量: 1
ID 1 本地端口 192.168.1.180:8808 远程端口 0.0.0.0:88 服务器uuid 44b5b521-719c-4e2b-b069-ad176d8d88ba
rakshasa\bind>close 1
rakshasa\bind>
```
