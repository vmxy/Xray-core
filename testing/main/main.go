package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/xtls/xray-core/app/proxyman"
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
	"github.com/xtls/xray-core/proxy/freedom"
	v2http "github.com/xtls/xray-core/proxy/http"
	"github.com/xtls/xray-core/testing/scenarios"
	v2httptest "github.com/xtls/xray-core/testing/servers/http"
	"github.com/xtls/xray-core/testing/servers/tcp"
)

var serverFlags = flag.NewFlagSet("server", flag.ExitOnError)
var clientFlags = flag.NewFlagSet("client", flag.ExitOnError)

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
	/* fmt.Println("main args mode ", os.Args[1])
	mode := os.Args[1]
	switch mode {
	case "server":
	case "client":
	case "proxy":
		StartProxy()
	} */
	StartProxy()

}
func StartProxy() {
	log.Println("start proxy")
	// 创建基础配置 转换为 Xray 配置
	coreConfig, err := NewConfig().Build()
	if err != nil {
		log.Fatalf("Failed to build config: %v", err)
	}
	log.Println("core.New config")
	// 创建并启动实例
	instance, err := core.New(coreConfig)
	if err != nil {
		log.Fatalf("Failed to create instance: %v", err)
	}

	if err := instance.Start(); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
	defer instance.Close()

	log.Println("Proxy server started")
	log.Println("Press Ctrl+C to stop")
	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutting down...")
}
func TestHttpConformance() {
	httpServerPort := tcp.PickPort()
	httpServer := &v2httptest.Server{
		Port:        httpServerPort,
		PathHandler: make(map[string]http.HandlerFunc),
	}
	_, err := httpServer.Start()
	fmt.Println("http server port ", httpServerPort)
	common.Must(err)
	defer httpServer.Close()

	proxyPort := tcp.PickPort()
	fmt.Println("proxy server port ", proxyPort)
	proxyServerConfig := &core.Config{
		Inbound: []*core.InboundHandlerConfig{
			{
				ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(proxyPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				}),
				ProxySettings: serial.ToTypedMessage(&v2http.ServerConfig{}),
			},
		},
		Outbound: []*core.OutboundHandlerConfig{
			{
				ProxySettings: serial.ToTypedMessage(&freedom.Config{}),
			},
		},
	}

	proxyServers, err := scenarios.InitializeServerConfigs(proxyServerConfig)
	common.Must(err)
	defer scenarios.CloseAllServers(proxyServers)

	{
		/// 模拟发起http请求
		transport := &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse("http://127.0.0.1:" + proxyPort.String())
			},
		}

		client := &http.Client{
			Transport: transport,
		}

		resp, err := client.Get("http://127.0.0.1:" + httpServerPort.String())
		common.Must(err)
		if resp.StatusCode != 200 {
			fmt.Println("status: ", resp.StatusCode)
		}

		content, err := io.ReadAll(resp.Body)
		common.Must(err)
		if string(content) != "Home" {

		}
		fmt.Println("------body: ", string(content))
	}
	select {}
}

func TestHttpError() {
	tcpServer := tcp.Server{
		MsgProcessor: func(msg []byte) []byte {
			return []byte{}
		},
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	time.AfterFunc(time.Second*2, func() {
		tcpServer.ShouldClose = true
	})

	serverPort := tcp.PickPort()
	serverConfig := &core.Config{
		Inbound: []*core.InboundHandlerConfig{
			{
				ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				}),
				ProxySettings: serial.ToTypedMessage(&v2http.ServerConfig{}),
			},
		},
		Outbound: []*core.OutboundHandlerConfig{
			{
				ProxySettings: serial.ToTypedMessage(&freedom.Config{}),
			},
		},
	}

	servers, err := scenarios.InitializeServerConfigs(serverConfig)
	common.Must(err)
	defer scenarios.CloseAllServers(servers)

	{
		transport := &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse("http://127.0.0.1:" + serverPort.String())
			},
		}

		client := &http.Client{
			Transport: transport,
		}

		resp, err := client.Get("http://127.0.0.1:" + dest.Port.String())
		if resp != nil && resp.StatusCode != 503 || err != nil && !strings.Contains(err.Error(), "malformed HTTP status code") {
			fmt.Println("should not receive http response", err)
		}
	}
}

func TestHTTPConnectMethod() {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	serverPort := tcp.PickPort()
	serverConfig := &core.Config{
		Inbound: []*core.InboundHandlerConfig{
			{
				ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				}),
				ProxySettings: serial.ToTypedMessage(&v2http.ServerConfig{}),
			},
		},
		Outbound: []*core.OutboundHandlerConfig{
			{
				ProxySettings: serial.ToTypedMessage(&freedom.Config{}),
			},
		},
	}

	servers, err := scenarios.InitializeServerConfigs(serverConfig)
	common.Must(err)
	defer scenarios.CloseAllServers(servers)

	{
		transport := &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse("http://127.0.0.1:" + serverPort.String())
			},
		}

		client := &http.Client{
			Transport: transport,
		}

		payload := make([]byte, 1024*64)
		common.Must2(rand.Read(payload))

		ctx := context.Background()
		req, err := http.NewRequestWithContext(ctx, "Connect", "http://"+dest.NetAddr()+"/", bytes.NewReader(payload))
		req.Header.Set("X-a", "b")
		req.Header.Set("X-b", "d")
		common.Must(err)

		resp, err := client.Do(req)
		common.Must(err)
		if resp.StatusCode != 200 {
			fmt.Println("status: ", resp.StatusCode)
		}

		content := make([]byte, len(payload))
		common.Must2(io.ReadFull(resp.Body, content))
		if r := cmp.Diff(content, xor(payload)); r != "" {
			fmt.Println(r)
		}
	}
}

func TestHttpPost() {
	httpServerPort := tcp.PickPort()
	httpServer := &v2httptest.Server{
		Port: httpServerPort,
		PathHandler: map[string]http.HandlerFunc{
			"/testpost": func(w http.ResponseWriter, r *http.Request) {
				payload, err := buf.ReadAllToBytes(r.Body)
				r.Body.Close()

				if err != nil {
					w.WriteHeader(500)
					w.Write([]byte("Unable to read all payload"))
					return
				}
				payload = xor(payload)
				w.Write(payload)
			},
		},
	}

	_, err := httpServer.Start()
	common.Must(err)
	defer httpServer.Close()

	serverPort := tcp.PickPort()
	serverConfig := &core.Config{
		Inbound: []*core.InboundHandlerConfig{
			{
				ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				}),
				ProxySettings: serial.ToTypedMessage(&v2http.ServerConfig{}),
			},
		},
		Outbound: []*core.OutboundHandlerConfig{
			{
				ProxySettings: serial.ToTypedMessage(&freedom.Config{}),
			},
		},
	}

	servers, err := scenarios.InitializeServerConfigs(serverConfig)
	common.Must(err)
	defer scenarios.CloseAllServers(servers)

	{
		transport := &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse("http://127.0.0.1:" + serverPort.String())
			},
		}

		client := &http.Client{
			Transport: transport,
		}

		payload := make([]byte, 1024*64)
		common.Must2(rand.Read(payload))

		resp, err := client.Post("http://127.0.0.1:"+httpServerPort.String()+"/testpost", "application/x-www-form-urlencoded", bytes.NewReader(payload))
		common.Must(err)
		if resp.StatusCode != 200 {
			fmt.Println("status: ", resp.StatusCode)
		}

		content, err := io.ReadAll(resp.Body)
		common.Must(err)
		if r := cmp.Diff(content, xor(payload)); r != "" {
			fmt.Println(r)
		}
	}
}
func xor(b []byte) []byte {
	r := make([]byte, len(b))
	for i, v := range b {
		r[i] = v ^ 'c'
	}
	return r
}
func setProxyBasicAuth(req *http.Request, user, pass string) {
	req.SetBasicAuth(user, pass)
	req.Header.Set("Proxy-Authorization", req.Header.Get("Authorization"))
	req.Header.Del("Authorization")
}

func TestHttpBasicAuth() {
	httpServerPort := tcp.PickPort()
	httpServer := &v2httptest.Server{
		Port:        httpServerPort,
		PathHandler: make(map[string]http.HandlerFunc),
	}
	_, err := httpServer.Start()
	common.Must(err)
	defer httpServer.Close()

	serverPort := tcp.PickPort()
	serverConfig := &core.Config{
		Inbound: []*core.InboundHandlerConfig{
			{
				ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
					PortList: &net.PortList{Range: []*net.PortRange{net.SinglePortRange(serverPort)}},
					Listen:   net.NewIPOrDomain(net.LocalHostIP),
				}),
				ProxySettings: serial.ToTypedMessage(&v2http.ServerConfig{
					Accounts: map[string]string{
						"a": "b",
					},
				}),
			},
		},
		Outbound: []*core.OutboundHandlerConfig{
			{
				ProxySettings: serial.ToTypedMessage(&freedom.Config{}),
			},
		},
	}

	servers, err := scenarios.InitializeServerConfigs(serverConfig)
	common.Must(err)
	defer scenarios.CloseAllServers(servers)

	{
		transport := &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse("http://127.0.0.1:" + serverPort.String())
			},
		}

		client := &http.Client{
			Transport: transport,
		}

		{
			resp, err := client.Get("http://127.0.0.1:" + httpServerPort.String())
			common.Must(err)
			if resp.StatusCode != 407 {
				fmt.Println("status: ", resp.StatusCode)
			}
		}

		{
			ctx := context.Background()
			req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:"+httpServerPort.String(), nil)
			common.Must(err)

			setProxyBasicAuth(req, "a", "c")
			resp, err := client.Do(req)
			common.Must(err)
			if resp.StatusCode != 407 {
				fmt.Println("status: ", resp.StatusCode)
			}
		}

		{
			ctx := context.Background()
			req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:"+httpServerPort.String(), nil)
			common.Must(err)

			setProxyBasicAuth(req, "a", "b")
			resp, err := client.Do(req)
			common.Must(err)
			if resp.StatusCode != 200 {
				fmt.Println("status: ", resp.StatusCode)
			}

			content, err := io.ReadAll(resp.Body)
			common.Must(err)
			if string(content) != "Home" {
				fmt.Println("body: ", string(content))
			}
		}
	}
}
