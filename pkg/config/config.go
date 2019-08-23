// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package config

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/koding/multiconfig"

	"openpitrix.io/newbilling/pkg/constants"
	"openpitrix.io/newbilling/pkg/logger"
)

type Config struct {
	Log LogConfig

	Etcd struct {
		Endpoints string `default:"127.0.0.1:2379"`
	}

	App struct {
		Host string `default:"localhost"`
		Port string `default:"8081"`

		ApiHost string `default:"localhost"`
		ApiPort string `default:"8080"`
	}
}

var instance *Config

var once sync.Once

func GetInstance() *Config {
	once.Do(func() {
		instance = &Config{}
	})
	return instance
}

type LogConfig struct {
	Level string `default:"debug"` // debug, info, warn, error, fatal
}

func (c *Config) PrintUsage() {
	fmt.Fprintf(os.Stdout, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprint(os.Stdout, "\nSupported environment variables:\n")
	e := newLoader(constants.ServiceName)
	e.PrintEnvs(new(Config))
	fmt.Println("")
}

func (c *Config) GetFlagSet() *flag.FlagSet {
	flag.CommandLine.Usage = c.PrintUsage
	return flag.CommandLine
}

func (c *Config) ParseFlag() {
	c.GetFlagSet().Parse(os.Args[1:])
}

func (c *Config) LoadConf() *Config {
	c.ParseFlag()
	config := instance

	m := &multiconfig.DefaultLoader{}
	m.Loader = multiconfig.MultiLoader(newLoader(constants.ServiceName))
	m.Validator = multiconfig.MultiValidator(
		&multiconfig.RequiredValidator{},
	)
	err := m.Load(config)
	if err != nil {
		panic(err)
	}

	logger.Info(nil, "LoadConf: %+v", config)

	return config
}
