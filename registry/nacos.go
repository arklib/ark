package registry

import (
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/clients"
	client "github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func NewNacosClient(c *Config) (cli client.INamingClient, err error) {
	var srvConfigs []constant.ServerConfig

	for _, addr := range c.Addrs {
		parts := strings.Split(addr, ":")

		port := uint64(8848)
		if len(addr) == 2 {
			port, err = strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				return
			}
		}

		sc := *constant.NewServerConfig(parts[0], port)
		srvConfigs = append(srvConfigs, sc)
	}

	// default namespace
	if c.Namespace == "" {
		c.Namespace = "public"
	}

	cliConfig := constant.ClientConfig{
		NamespaceId:         c.Namespace,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              c.LogDir,
		CacheDir:            c.CacheDir,
		LogLevel:            "error",
		Username:            c.Username,
		Password:            c.Password,
	}

	return clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &cliConfig,
			ServerConfigs: srvConfigs,
		},
	)
}
