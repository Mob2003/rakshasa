[主菜单](./cli.md)

## yaml文件例子，保存在启动目录下

```
dstnode:
- 192.168.1.180:8883
password: ""
port: 8883
listenip:
- 192.168.1.151
limit: false
filename: config.yaml
```

#### 如果有启动参数将会覆盖掉yaml配置，如-d会覆盖掉dstnode

- dstnode 目标服务器   对应启动参数：-d
- password 传输秘钥    对应启动参数：-password
- port 本节点监听端口 对应启动参数：-p
- listenip 本节点监听ip 其他节点断线后，将会尝试连接此ip与上述端口，通过使用-limit来关闭额外连接功能
- limit 为true时候，除了目标服务器不会进行额外连接。默认为false，节点断线后，将会自动尝试连接所有以连接过的节点ip port。
- filename 配置文件名字，控制台save时候保存

## shell使用说明

### save

 修改的内容不会立刻写入文件，必须通过执行save之后才会保存到filename

### d

 修改yaml的dstnode，使用方法 d 192.168.1.1:8883,192.168.1.2:8883

### password

 修改yaml的password，使用方法 password "asdlkj" 解析双引号里面内容，结果不带双引号，参数可以不包含双引号，如果需要使用双引号请输入\\"

### port

修改yaml的port

### ip

修改yaml的listenip,可以配置多个ip，以,隔开

### limit

修改yaml的limit

### f

修改yaml的filename



```shell
rakshasa>config
rakshasa\config>help

Commands:
  clear         clear the screen
  d             修改上级节点地址，格式为 ip:端口 多个节点以,隔开 注意：不会立刻连接设置节点， 当发生 节点掉线重连 时候会连接该地址
  exit          exit the program
  f             修改配置文件名，使用方法 f config.yaml
  help          display help
  info          打印当前配置
  ip            修改本节点连接ip，当其他节点进行额外连接时候，优先使用此ip连接, 多个ip以,隔开
  limit         修改本节点Limit设置，使用方法 limit true
  new           与一个或者多个节点连接，使用方法 new ip:端口 多个地址以,间隔 如1080 127.0.0.1:1081,127.0.0.1:1082
  password      修改通讯密码,立即生效
  ping          ping 节点
  port          修改监听端口,立即生效
  print         列出所有节点
  save          保存文件


rakshasa\config>
```
