package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
)

func main() {
	all := flag.Bool("nocli", false, "生成新的证书，使用随机的种子生成控制端节点和普通节点")
	gencert := flag.Bool("gencert", false, "生成证书")
	fullNode := flag.Bool("fullnode", false, "只编译生成控制端节点+普通节点")
	node := flag.Bool("node", false, "只生成普通节点")
	flag.Parse()
	if *all == true {
		*gencert = true
		*fullNode = true
		*node = false
	}
	if *gencert {
		if runtime.GOOS == "windows" {
			_, err := exec.Command("cmd.exe", "/c", "cd gencert && go run main.go").CombinedOutput()
			if err != nil {
				log.Fatal(`无法生成证书，请手动执行"cd gencert && go run main.go"`)
				return
			}
		} else {
			_, err := exec.Command("bin/bash", "-c", "cd gencert && go run main.go").CombinedOutput()
			if err != nil {
				log.Fatal(`无法生成证书，请手动执行"cd gencert && go run main.go"`)
				return
			}
		}
	}
	if b, _ := os.ReadFile("./cert/private.pem"); len(b) == 0 {
		log.Fatal(`无法生成证书，请手动执行"cd gencert && go run main.go"`)
		return
	}
	if b, _ := os.ReadFile("./cert/public.pem"); len(b) == 0 {
		log.Fatal(`无法生成证书，请手动执行"cd gencert && go run main.go"`)
		return
	}
	if b, _ := os.ReadFile("./cert/server.crt"); len(b) == 0 {
		log.Fatal(`无法生成证书，请手动执行"cd gencert && go run main.go"`)
		return
	}
	if b, _ := os.ReadFile("./cert/server.key"); len(b) == 0 {
		log.Fatal(`无法生成证书，请手动执行"cd gencert && go run main.go"`)
		return
	}
	if *fullNode {
		b, _ := os.ReadFile("./cert/private.pem")
		data := fmt.Sprintf("package cert\r\n func init(){\r\nprivateKey=%#v\r\n}\r\n", b)
		os.WriteFile("./cert/private.go", []byte(data), 0655)
		buildNode("build_fullnode")
	} else if *node {
		os.Remove("./cert/private.go")
		buildNode("build_node")
	} else {
		flag.PrintDefaults()

	}
}
func buildNode(name string) {
	if runtime.GOOS == "windows" {
		cmdstr := fmt.Sprintf(`cd build && %s.bat`, name)
		_, err := exec.Command("cmd.exe", "/c", cmdstr).CombinedOutput()
		if err != nil {
			log.Fatal(`无法编译,请手动执行 ` + cmdstr)
			return
		}

	} else {
		exec.Command("bin/bash", "-c", "cd build && sudo chmod 755 "+name+".sh").CombinedOutput()
		cmdstr := fmt.Sprintf(`cd build && ./%s.sh`, name)
		_, err := exec.Command("/bin/bash", "-c", cmdstr).CombinedOutput()
		if err != nil {
			log.Fatal(`无法编译,请手动执行 ` + cmdstr)
			return
		}
	}
}
