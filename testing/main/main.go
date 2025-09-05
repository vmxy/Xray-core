package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

var serverFlags = flag.NewFlagSet("server", flag.ExitOnError)
var clientFlags = flag.NewFlagSet("client", flag.ExitOnError)

const message = `这是第一行
这是第二行
这是第三行`

func init() {
	// 创建新的 FlagSet
	// 为每个 FlagSet 定义不同的标志
	serverPort := serverFlags.Int("port", 8080, "Server port")
	clientHost := clientFlags.String("host", "localhost", "Server host")

	if len(os.Args) < 2 {
		fmt.Println("Expected 'server' or 'client' subcommand")
		os.Exit(1)
	}

	// 根据子命令解析不同的标志集
	switch os.Args[1] {
	case "server":
		serverFlags.Parse(os.Args[2:])
		fmt.Println("Server port:", *serverPort)
	case "client":
		clientFlags.Parse(os.Args[2:])
		fmt.Println("Connecting to:", *clientHost)
	case "proxy":
		clientFlags.Parse(os.Args[2:])
		fmt.Println("create proxy:", *clientHost)
	default:
		fmt.Println("Unknown subcommand:", os.Args[1])
		//os.Exit(1)
	}
	//flag.PrintDefaults()
}
func main() {
	fmt.Println("main args mode ", os.Args[1])
	mode := os.Args[1]
	switch mode {
	case "server":
		StartProxyServer()
	case "client":
		StartProxyClient()
	default:
		StartProxyClient()
	}
}
func StartProxyClient() {
	cfg := loadClientConfig()
	//cfg := buildClientConfig()
	// 实例化 core
	instance, err := core.New(cfg)
	if err != nil {
		log.Fatal("failed to create core instance:", err)
	}

	if err := instance.Start(); err != nil {
		log.Fatal("failed to start core:", err)
	}

	select {} // 阻塞，保持进程运行
}
func StartProxyServer() {
	//cfg := loadConfig()
	cfg := loadServerConfig()
	// 实例化 core
	instance, err := core.New(cfg)
	if err != nil {
		log.Fatal("failed to create core instance:", err)
	}

	if err := instance.Start(); err != nil {
		log.Fatal("failed to start core:", err)
	}

	select {} // 阻塞，保持进程运行
}
