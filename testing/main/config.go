package main

import (
	"encoding/json"

	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/infra/conf"
	v2http "github.com/xtls/xray-core/proxy/http"
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
