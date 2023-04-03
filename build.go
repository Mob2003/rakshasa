package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
)

func main() {
	all := flag.Bool("all", false, "生成新的证书，生成控制端节点和普通节点")
	nocert := flag.Bool("nocert", false, "不生成证书")
	gencert := flag.Bool("gencert", false, "生成证书")
	fullNode := flag.Bool("fullnode", false, "编译生成控制端节点+普通节点")
	node := flag.Bool("node", false, "只生成普通节点")
	fullNodeLite := flag.Bool("fullnode-lite", false, "编译生成lite版本的： 控制端节点+普通节点")
	nodeLite := flag.Bool("node-lite", false, "只生成lite版本的： 普通节点")
	genfullNodePrivate := flag.Bool("gen-private", false, "将private.pem转为private.go")
	flag.Parse()
	if *all == true {
		*gencert = true
		*fullNode = true
		*node = true
		*fullNodeLite = true
		*nodeLite = true
	}
	if *nocert {
		*gencert = false
	}
	if *genfullNodePrivate {
		b, _ := os.ReadFile("./cert/private.pem")
		data := fmt.Sprintf("package cert\r\n func init(){\r\nprivateKey=%#v\r\n}\r\n", b)
		os.WriteFile("./cert/private.go", []byte(data), 0655)
		return
	}
	if *gencert {
		if runtime.GOOS == "windows" {
			_, err := exec.Command("cmd.exe", "/c", "cd gencert && go run main.go").CombinedOutput()
			if err != nil {
				log.Fatal(`无法生成证书，请手动执行"cd gencert && go run main.go"`)
				return
			}
		} else {
			_, err := exec.Command("/bin/bash", "-c", "cd gencert && go run main.go").CombinedOutput()
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
	}
	if *fullNodeLite {
		b, _ := os.ReadFile("./cert/private.pem")
		data := fmt.Sprintf("package cert\r\n func init(){\r\nprivateKey=%#v\r\n}\r\n", b)
		os.WriteFile("./cert/private.go", []byte(data), 0655)
		buildNode("build_fullnode_lite")
	} else if *nodeLite {
		os.Remove("./cert/private.go")
		buildNode("build_node_lite")
	}
	if !*gencert && !*fullNode && !*node && !*fullNodeLite && !*nodeLite && !*genfullNodePrivate {
		flag.PrintDefaults()
	}
}
func buildNode(name string) {
	if runtime.GOOS == "windows" {
		cmdstr := fmt.Sprintf(`cd build && %s.bat`, name)
		err := runCommand("cmd.exe", "/c", cmdstr)
		if err != nil {
			log.Fatal(`无法编译,请手动执行 ` + cmdstr)
			return
		}
	} else {
		exec.Command("/bin/bash", "-c", "cd build && sudo chmod 755 "+name+".sh").CombinedOutput()
		cmdstr := fmt.Sprintf(`cd build && ./%s.sh`, name)
		err := runCommand("/bin/bash", "-c", cmdstr)
		if err != nil {
			log.Fatal(`无法编译,请手动执行 ` + cmdstr)
			return
		}

	}
}
func runCommand(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	stdout, err := cmd.StdoutPipe()
	err = cmd.Start()
	if err != nil {
		return err
	}

	//创建一个流来读取管道内内容，这里逻辑是通过一行一行的读取的
	reader := bufio.NewReader(stdout)
	//实时循环读取输出流中的一行内容
	go func() {

		for {
			line, err2 := reader.ReadString('\n')
			if err2 != nil || io.EOF == err2 {
				break
			}

			fmt.Print(line)
		}

	}()
	cmd.Wait()
	return err
}
