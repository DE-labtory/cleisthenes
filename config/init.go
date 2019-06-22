package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

func Init(configPath string) error {
	if configPath != "" {
		conf, err := readConfigFile(configPath)
		if err != nil {
			return err
		}
		return writeConfigFile(conf)
	}

	return writeConfigFile(defaultConfig)
}

// readConfigFile reads the config from `filename` into `cfg`.
func readConfigFile(filename string) (*Config, error) {
	conf := &Config{}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := yaml.NewDecoder(f).Decode(conf); err != nil {
		return nil, fmt.Errorf("failure to decode config: %s", err)
	}
	return conf, nil
}

// writeConfigFile writes the config from `cfg` into `filename`.
func writeConfigFile(cfg *Config) error {
	err := os.MkdirAll(filepath.Dir(configPath), 0775)
	if err != nil {
		return err
	}

	if fileExists(configPath) {
		if err := os.Remove(configPath); err != nil {
			return err
		}
	}

	f, err := openFile(configPath, 0660)
	if err != nil {
		return err
	}
	defer f.Close()

	return encode(f, cfg)
}

// encode configuration with JSON
func encode(w io.Writer, value interface{}) error {
	buf, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

// File behaves like os.File, but does an atomic rename operation at Close.
type file struct {
	*os.File
	path string
}

func openFile(path string, mode os.FileMode) (*file, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(f.Name(), mode); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	return &file{File: f, path: path}, nil
}

// fileExists check if the file with the given path exits.
func fileExists(filename string) bool {
	fi, err := os.Lstat(filename)
	if fi != nil || (err != nil && !os.IsNotExist(err)) {
		return true
	}
	return false
}
