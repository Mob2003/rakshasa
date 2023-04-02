#! /bin/sh
cd ../

file=./cert/private.go
 
if [ ! -f "$file" ]; then
	cd rakshasa_lite
echo "Start build node-lite"
echo "build windows"
GOOS=windows go build -a -ldflags="-w -s" -trimpath -o ../bin/rakshasa_node_lite_amd64_win.exe main.go
echo "build linux"
GOOS=linux go build -a -ldflags="-w -s" -trimpath -o ../bin/rakshasa_node_lite_amd64_linux main.go
echo "build darwin"
GOOS=darwin go build -a -ldflags="-w -s" -trimpath -o ../bin/rakshasa_node_lite_amd64_darwin main.go
else
    echo "请删除private.go后在编译"
fi