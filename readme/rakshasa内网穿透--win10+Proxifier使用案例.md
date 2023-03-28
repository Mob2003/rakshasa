
### 作者: Mob2003
rakshasa开源地址
```url
https://github.com/Mob2003/rakshasa
```

### 前言

在渗透过程中，我们需要快速实现内网穿透，从而绕过网络访问限制，直接从外网来访问内网，这篇文章介绍一下，windows使用《rakshasa》如何利用socks5访问内网3389远程桌面。




### 实验环境搭建

如下图，实验目的，192.168.2.2使用远程桌面连到192.168.3.3

![image](https://user-images.githubusercontent.com/128351726/228122343-2b9ac594-a11e-4a33-8982-db66f0ddeec0.png)

#### 裸机测试

当我们从192.168.2.2去访问192.168.3.3的远程桌面，不出意外，提示远程桌面无法连接

![image](https://user-images.githubusercontent.com/128351726/228122469-fd88383d-e3f2-4883-8df1-8eea1f06133a.png)

### rakshasa节点连通

注：请更新最新版本rakshasa

- 192.168.2.2 连接到192.168.2.1
  
  ```shell
  .\rakshasa.exe -d 192.168.2.1:8883
  ```
- 192.168.3.2 连接到192.168.3.1
  
  ```shell
  .\rakshasa.exe -d 192.168.3.1:8883
  ```
  
  在CLI上print，已经看到机器已经上线
  ![image](https://user-images.githubusercontent.com/128351726/228122549-12305b00-7617-40d9-a35e-af85424ea08b.png)



### 启动socks5正向代理

代理方向192.168.2.2代理到192.168.3.2，使用命令行如下

```shell
 .\rakshasa.exe -d 192.168.2.1:8883,192.168.3.1:8883,192.168.3.2:8883 -socks5 1080
```

命令解析：

- **-d** 连接到这几个节点，并且按照顺序从左到右连接，192.168.2.1->192.168.3.1->192.168.3.2，节点可以是未连接状态

- **-socks5 1080** 本地开启一个socks5代理，流量出口在-d最右边一个节点，本命令中，socks5代理是从本机开启，最终出口是192.168.3.2这台机器
  
  

特殊用法：

   先连接到192.168.2.1:8883，再从192.168.2.1:8883连接到f2be7b68-cc22-4506-88b1-69a99081f57d，节点必须是已连接状态

```shell
.\rakshasa.exe -d 192.168.2.1:8883,f2be7b68-cc22-4506-88b1-69a99081f57d -socks5 1080
```
![image](https://user-images.githubusercontent.com/128351726/228122782-a6a14cc8-5d08-461e-abb7-1d389727bf66.png)

### Proxifier配置
打开代理设置，把rakshasa.exe设置为直连
![image](https://user-images.githubusercontent.com/128351726/228122858-c0cf1840-5fb7-48bf-aedd-d8488cf7d192.png)
添加socks5代理
![image](https://user-images.githubusercontent.com/128351726/228122902-2dfc1c4b-a61c-4abb-bf32-58c4b65299a8.png)
### 连接远程桌面
直接打开mstsc.exe，输入192.168.3.3，发现已经可以通过本地socks5代理udp过去了
![image](https://user-images.githubusercontent.com/128351726/228123026-3ef1711e-afd2-4abb-b232-45473b6c6136.png)
