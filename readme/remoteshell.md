[主菜单](./cli.md)

### file
使用方法file 节点，如file 1，进入节点后可用cd,dir,upload,download命令
- cd 切换远程工作目录
- dir 列出当前目录下文件和文件夹信息
- upload 上传文件，两个参数，参数二可为空，用法:upload 本地文件目录 远程目录（为空传到工作目录）
- download 下载文件，两个参数，参数二可为空，用法:download 远程文件 本地目录(为空本地执行目录)

```shell
rakshasa>remoteshell
rakshasa\remoteshell>file
参数错误
rakshasa\remoteshell>print
ID  UUID                                  HostName                GOOS          IP                       listenIP
-----------------------------------------------------------------------------------------------------------------------------
 1  12a8f492-5fff-4b75-935a-533a276d546e  DESKTOP-DAAI4F1         windows x64  (localhost):8883
 2  44b5b521-719c-4e2b-b069-ad176d8d88ba  DESKTOP-DAAI4F1         windows x64  192.168.1.137:8884
rakshasa\remoteShell>file 2
4b5b521-719c-4e2b-b069-ad176d8d88ba d:/>help

Commands:
  cd            切换工作目录
  clear         clear the screen
  dir           打印当前目录文件
  download      下载文件 ，download 远程文件 本地目录(为空本地执行目录)
  exit          exit the program
  help          display help
  upload        上传文件 ，upload 本地文件 远程目录(为空传到工作目录)


44b5b521-719c-4e2b-b069-ad176d8d88ba d:/>
```

### shell
使用方法shell 节点 启动参数，如shell 1 powershell。启动参数可为空

windows下默认启动cmd

linux下默认启动bash，如启动失败可尝试改为/bin/sh或者/bin/zsh等

```shell
rakshasa\remoteshell>shell 2 powershell
Windows PowerShell
版权所有 (C) Microsoft Corporation。保留所有权利。

尝试新的跨平台 PowerShell https://aka.ms/pscore6


PS D:\>  whoami
desktop-daai4f1\administrator

PS D:\> exit
请按回车键退出
rakshasa\remoteshell>
```