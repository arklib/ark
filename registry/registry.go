package registry

import (
	"github.com/cloudwego/kitex/pkg/discovery"
	"github.com/cloudwego/kitex/pkg/registry"
	etcd "github.com/kitex-contrib/registry-etcd"
	nacosRegistry "github.com/kitex-contrib/registry-nacos/registry"
	nacosResolver "github.com/kitex-contrib/registry-nacos/resolver"

	"github.com/arklib/ark/errx"
)

type Config struct {
	Driver    string
	Addrs     []string
	Namespace string
	Username  string
	Password  string
	LogDir    string // nacos log dir
	CacheDir  string // nacos cache dir
}

func NewResolver(c *Config) (discovery.Resolver, error) {
	switch c.Driver {
	case "etcd":
		return etcd.NewEtcdResolver(c.Addrs)
	case "nacos":
		nacos, err := NewNacosClient(c)
		if err != nil {
			return nil, err
		}
		r := nacosResolver.NewNacosResolver(nacos)
		return r, nil
	default:
		err := errx.Sprintf("unknown driver: %s", c.Driver)
		return nil, err
	}
}

func NewRegistry(c *Config) (registry.Registry, error) {
	switch c.Driver {
	case "etcd":
		return etcd.NewEtcdRegistry(c.Addrs)
	case "nacos":
		nacos, err := NewNacosClient(c)
		if err != nil {
			return nil, err
		}
		r := nacosRegistry.NewNacosRegistry(nacos)
		return r, nil
	default:
		err := errx.Sprintf("unknown driver: %s", c.Driver)
		return nil, err
	}
}
