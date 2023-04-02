#! /bin/sh
cd ../

file=./cert/private.go
 
if [ -f "$file" ]; then
	cd rakshasa_lite
echo "Start build fullnode-lite"
echo "build windows"
GOOS=windows go build -a -ldflags="-w -s" -trimpath -o ../bin/rakshasa_fullnode_amd64_lite_win.exe main.go
echo "build linux"
GOOS=linux go build -a -ldflags="-w -s" -trimpath -o ../bin/rakshasa_fullnode_amd64_lite_linux main.go
echo "build darwin"
GOOS=darwin go build -a -ldflags="-w -s" -trimpath -o ../bin/rakshasa_fullnode_amd64_lite_darwin main.go
cd ../cert
rm -f "private.go"
cd ../build
chmod 755 build_node.sh
./build_node.sh
else
    echo "找不到private.go,请使用 go run build.go -fullnode-lite来编译"
fi