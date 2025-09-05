package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/xtls/xray-core/common/cmdarg"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/infra/conf"
	v2http "github.com/xtls/xray-core/proxy/http"

	"github.com/xtls/xray-core/core"
)

func NewConfig() *conf.Config {
	address := conf.Address{
		Address: xnet.ParseAddress("127.0.0.1"),
	}
	// 创建基础配置
	config := &conf.Config{
		LogConfig: &conf.LogConfig{
			LogLevel:  "info", // 可选: "debug", "info", "warning", "error", "none"
			AccessLog: "",     // 访问日志路径，不填默认输出到 stdout
			ErrorLog:  "",     // 错误日志路径，不填默认输出到 stdout
		},
		InboundConfigs: []conf.InboundDetourConfig{
			{
				Protocol: "socks",
				ListenOn: &address,
				PortList: &conf.PortList{
					Range: []conf.PortRange{
						{
							From: 1091,
							To:   1091,
						},
					},
				},
			},
			{
				Protocol: "http",
				ListenOn: &address,
				PortList: &conf.PortList{
					Range: []conf.PortRange{
						{
							From: 1092,
							To:   1092,
						},
					},
				},
			},
		},
		OutboundConfigs: []conf.OutboundDetourConfig{
			{
				Protocol: "freedom",
				Tag:      "direct",
			},
			buildHttpOutBoundConfig(),
		},
	}
	return config
}

func buildHttpOutBoundConfig() conf.OutboundDetourConfig {

	var account = serial.ToTypedMessage(&v2http.Account{
		Username: "",
		Password: "",
	})

	// 构建 HTTP 客户端配置
	httpSettings := &v2http.ClientConfig{
		Server: []*protocol.ServerEndpoint{
			{
				Address: xnet.NewIPOrDomain(xnet.ParseAddress("127.0.0.1")),
				Port:    1080,
				User: []*protocol.User{
					{
						Account: account,
					},
				},
			},
		},
	}
	/* */
	// 将配置序列化为 JSON RawMessage
	settingsJson, err := json.Marshal(httpSettings)
	if err != nil {
		panic(err) // 在实际代码中应该更优雅地处理错误
	}
	//network := conf.TransportProtocol("tcp") // 先创建值

	// 返回完整的 OutboundDetourConfig
	return conf.OutboundDetourConfig{
		Protocol: "http",
		Tag:      "http-proxy", // 更具描述性的tag
		Settings: (*json.RawMessage)(&settingsJson),
		/* 		StreamSetting: &conf.StreamConfig{
			Network:  &network,
			Security: "none",
			TLSSettings: &conf.TLSConfig{
				ServerName: "go.x.one",
				Insecure:   true,
			},
		}, */
	}
}

func buildClientConfig() *core.Config {
	address := conf.Address{
		Address: xnet.ParseAddress("127.0.0.1"),
	}
	/* 	var account = serial.ToTypedMessage(&v2http.Account{
		Username: "",
		Password: "",
	}) */

	socketInboundSettings := &conf.SocksServerConfig{
		AuthMethod: "noauth",
		Accounts:   []*conf.SocksAccount{},
		UDP:        true,
		Host:       &address,
	}
	socketInboundSettingsJson, _ := json.Marshal(socketInboundSettings)

	// inbound: socks5
	socksInbound := &conf.InboundDetourConfig{
		Protocol: "socks",
		ListenOn: &address,
		PortList: &conf.PortList{
			Range: []conf.PortRange{{From: 1091, To: 1091}},
		},
		Settings: (*json.RawMessage)(&socketInboundSettingsJson),
	}

	httpInboundSettings := &conf.SocksServerConfig{
		AuthMethod: "noauth",
		Accounts:   []*conf.SocksAccount{},
	}
	httpInboundSettingsJson, _ := json.Marshal(httpInboundSettings)

	// inbound: http
	httpInbound := &conf.InboundDetourConfig{
		Protocol: "http",
		ListenOn: &address,
		PortList: &conf.PortList{
			Range: []conf.PortRange{{From: 1092, To: 1092}},
		},
		Settings: (*json.RawMessage)(&httpInboundSettingsJson),
	}

	// outbound: freedom
	freedomOutbound := &conf.OutboundDetourConfig{
		Protocol: "freedom",
		Tag:      "direct",
	}

	// outbound: http (free proxy, 127.0.0.1:1080)
	httpOutboundConfig := &conf.HTORemoteConfig{
		Address: &address,
		Port:    1080,
	}

	httpOutboundSettings := &conf.HTOClientConfig{
		Servers: []*conf.HTORemoteConfig{httpOutboundConfig},
	}
	settingsJson, _ := json.Marshal(httpOutboundSettings)
	network := conf.TransportProtocol("tcp")
	httpOutbound := &conf.OutboundDetourConfig{
		Protocol: "hto",
		Tag:      "proxy",
		Settings: (*json.RawMessage)(&settingsJson),
		StreamSetting: &conf.StreamConfig{
			Network:  &network,
			Security: "none",
		},
	}

	confConfig := &conf.Config{
		LogConfig: &conf.LogConfig{
			LogLevel: "info",
		},
		InboundConfigs:  []conf.InboundDetourConfig{*socksInbound, *httpInbound},
		OutboundConfigs: []conf.OutboundDetourConfig{*freedomOutbound, *httpOutbound},
	}

	// 转换为 core.Config
	coreCfg, err := confConfig.Build()
	if err != nil {
		log.Fatal("failed to build core.Config:", err)
	}
	return coreCfg
}

func loadClientConfig() *core.Config {
	arg := cmdarg.Arg{}
	arg.Set("config-client.jsonc")
	config, err := core.LoadConfig("json", arg)
	fmt.Println("load config ")
	if err != nil {
		log.Fatal(err)
	}
	return config
}
func loadServerConfig() *core.Config {
	jsonstr := `
{
    "log": {
        "loglevel": "info"
    },
    "inbounds": [
        {
            "listen": "127.0.0.1",
            "port": "1101",
            "protocol": "socks",
            "settings": {
                "auth": "password",
                "accounts": [
                    {
                        "user": "app",
                        "pass": "123456"
                    }
                ],
                "udp": true,
                "ip": "127.0.0.1"
            }
        },
        {
            "listen": "127.0.0.1",
            "port": "1102",
            "protocol": "http",
            "settings": {
                "auth": "password",
                "accounts": [
                    {
                        "user": "app",
                        "pass": "123456"
                    }
                ]
            }
        }
    ],
    "outbounds": [
		{
            "protocol": "socks",
            "settings": {
                "servers": [
                    {
                        "address": "127.0.0.1",
                        "port": 1080,
                        "users": []
                    }
                ]
            },
            "streamSettings": {
                "network": "tcp",
                "security": "none"
            },
            "tag": "proxy"
        },
        {
            "protocol": "freedom",
            "tag": "direct"
        },
        {
            "protocol": "blackhole",
            "tag": "block"
        }
    ]
}
`
	fmt.Println("jsonstr", jsonstr)
	cfg, _ := JsonToCoreConfig(jsonstr)
	return cfg
}
func JsonToCoreConfig(jsonStr string) (*core.Config, error) {
	// 1. 解析到中间层 conf.Config
	var raw conf.Config
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("解析 JSON 到 conf.Config 失败: %w", err)
	}

	// 2. 转换成 core.Config（protobuf 结构）
	cfg, err := raw.Build()
	if err != nil {
		return nil, fmt.Errorf("构建 core.Config 失败: %w", err)
	}

	return cfg, nil
}
func buildServerConfig() *core.Config {
	return nil
}
