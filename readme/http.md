[主菜单](./shell.md)

### new-httpproxy 

新建一个httpProxy连接，使用方法 new-httpproxy 配置字串符 目标服务器 代理池文件路径，如果远程节点为空，则为本地直接连
如 

new-httpproxy admin:123456@0.0.0.0:8080 127.0.0.1:8881,127.0.0.1:8882 out.txt
new-httpproxy 8080 out.txt



### list
打印出当前http监听情况

### close
关掉一个http监听

使用方法,先使用list获得当前执行的http的id，再执行 close id

```shell
rakshasa>httpproxy
rakshasa\httpproxy>new-httpproxy 8080
httpproxy start  :8080
本地httpProxy启动成功
rakshasa\httpproxy>list
当前连接数量: 1
ID 1 本地端口 :8080 转发服务器uuid 6268846f-f93e-4525-ad5c-ed6b3b773478
rakshasa\httpproxy>close 1
rakshasa\httpproxy>
```