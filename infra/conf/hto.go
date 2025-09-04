package conf

import (
	"encoding/json"

	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/proxy/hto"
	"google.golang.org/protobuf/proto"
)

type HTOAccount struct {
	Username string `json:"user"`
	Password string `json:"pass"`
}

func (v *HTOAccount) Build() *hto.Account {
	return &hto.Account{
		Username: v.Username,
		Password: v.Password,
	}
}

type HTOServerConfig struct {
	Accounts    []*HTOAccount `json:"accounts"`
	Transparent bool          `json:"allowTransparent"`
	UserLevel   uint32        `json:"userLevel"`
}

func (c *HTOServerConfig) Build() (proto.Message, error) {
	config := &hto.ServerConfig{
		AllowTransparent: c.Transparent,
		UserLevel:        c.UserLevel,
	}

	if len(c.Accounts) > 0 {
		config.Accounts = make(map[string]string)
		for _, account := range c.Accounts {
			config.Accounts[account.Username] = account.Password
		}
	}

	return config, nil
}

type HTORemoteConfig struct {
	Address *Address          `json:"address"`
	Port    uint16            `json:"port"`
	Users   []json.RawMessage `json:"users"`
}

type HTOClientConfig struct {
	Servers []*HTORemoteConfig `json:"servers"`
	Headers map[string]string  `json:"headers"`
}

func (v *HTOClientConfig) Build() (proto.Message, error) {
	config := new(hto.ClientConfig)
	config.Server = make([]*protocol.ServerEndpoint, len(v.Servers))
	for idx, serverConfig := range v.Servers {
		server := &protocol.ServerEndpoint{
			Address: serverConfig.Address.Build(),
			Port:    uint32(serverConfig.Port),
		}
		for _, rawUser := range serverConfig.Users {
			user := new(protocol.User)
			if err := json.Unmarshal(rawUser, user); err != nil {
				return nil, errors.New("failed to parse HTTP user").Base(err).AtError()
			}
			account := new(HTOAccount)
			if err := json.Unmarshal(rawUser, account); err != nil {
				return nil, errors.New("failed to parse HTTP account").Base(err).AtError()
			}
			user.Account = serial.ToTypedMessage(account.Build())
			server.User = append(server.User, user)
		}
		config.Server[idx] = server
	}
	config.Header = make([]*hto.Header, 0, 32)
	for key, value := range v.Headers {
		config.Header = append(config.Header, &hto.Header{
			Key:   key,
			Value: value,
		})
	}
	return config, nil
}
