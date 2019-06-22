package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/viper"
)

type HoneyBadger struct {
	NetworkSize int
	Byzantine   int
}

type Members struct {
	Addresses []string
}

type Tpke struct {
	MasterPublicKey string
	Threshold       int
}

// TODO: change default config
var defaultConfig = &Config{
	HoneyBadger: HoneyBadger{
		NetworkSize: 10,
		Byzantine:   3,
	},
	Members: Members{
		Addresses: []string{
			"localhost:8080",
			"localhost:8081",
		},
	},
	Tpke: Tpke{
		MasterPublicKey: "asdf",
		Threshold:       3,
	},
}

var once sync.Once

var configPath = os.Getenv("HOME") + "/.cleisthenes/config.yml"

type Config struct {
	HoneyBadger HoneyBadger
	Members     Members
	Tpke        Tpke
}

func Path() string {
	return configPath
}

func Get() *Config {
	once.Do(func() {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			panic("cannot read config")
		}
		err := viper.Unmarshal(&defaultConfig)
		if err != nil {
			panic(fmt.Sprintf("error in read config, err: %s", err))
		}
	})
	return defaultConfig
}
