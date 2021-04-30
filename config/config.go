package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

var c Config

type Config struct {
	RPC        string   `yaml:"rpc"`
	StartBlock uint64   `yaml:"startBlock"`
	Contract   string   `yaml:"contract"`
	Addresses  []string `yaml:"addresses"`
}

func (c *Config) ContainAddress(addr string) bool {
	for _, a := range c.Addresses {
		if strings.ToLower(addr) == a {
			return true
		}
	}
	return false
}

func GetConfig() *Config {
	return &c
}

func Init() error {
	fb, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		return fmt.Errorf("read config file fail, %v", err)
	}

	err = yaml.Unmarshal(fb, &c)
	if err != nil {
		return fmt.Errorf("parse config file fail, %v", err)
	}

	for i, a := range c.Addresses {
		c.Addresses[i] = strings.ToLower(a)
	}
	return nil
}
