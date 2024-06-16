package config

import (
	"log"

	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/json"
	"github.com/gookit/config/v2/toml"
	"github.com/gookit/config/v2/yaml"
)

type Config struct {
	*config.Config
}

func MustLoad(path string) *Config {
	c, err := Load(path)
	if err != nil {
		log.Fatal(err)
	}
	return c
}

func Load(path string) (c *Config, err error) {
	newConfig := config.New("ark").WithOptions(
		config.WithTagName("config"),
		config.ParseEnv,
		config.ParseTime,
		config.ParseDefault,
	)

	c = &Config{newConfig}
	c.AddDriver(json.Driver)
	c.AddDriver(toml.Driver)
	c.AddDriver(yaml.Driver)

	err = c.LoadFiles(path)
	return
}
