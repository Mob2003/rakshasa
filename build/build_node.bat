@echo off
cd ../

IF EXIST ./cert/private.go (
echo ÇëÉ¾³ýprivate.goºóÔÚ±àÒë
ping -n 3 127.0.0.1 > nul
) ELSE (
echo Start build node
echo build windows
set GOOS=windows
go build -a -ldflags="-w -s" -trimpath -o ./bin/rakshasa_node_amd64_win.exe main.go
echo build linux
set GOOS=linux
go build -a -ldflags="-w -s" -trimpath -o ./bin/rakshasa_node_amd64_linux main.go
echo build darwin
set GOOS=darwin
go build -a -ldflags="-w -s" -trimpath -o ./bin/rakshasa_node_amd64_darwin main.go
echo End

) 
