// Copyright 2019 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package global

import (
	"strings"
	"sync"

	"github.com/google/gops/agent"

	"openpitrix.io/scheduler/pkg/config"
	"openpitrix.io/scheduler/pkg/constants"
	"openpitrix.io/scheduler/pkg/etcd"
	"openpitrix.io/scheduler/pkg/logger"
)

type GlobalCfg struct {
	cfg  *config.Config
	etcd *etcd.Etcd
}

var instance *GlobalCfg
var once sync.Once

func GetInstance() *GlobalCfg {
	once.Do(func() {
		instance = newGlobalCfg()
	})
	return instance
}

func newGlobalCfg() *GlobalCfg {
	cfg := config.GetInstance().LoadConf()
	g := &GlobalCfg{cfg: cfg}

	g.setLoggerLevel()
	g.openEtcd()

	if err := agent.Listen(agent.Options{
		ShutdownCleanup: true,
	}); err != nil {
		logger.Critical(nil, "Failed to start gops agent")
	}
	return g
}

func (g *GlobalCfg) openEtcd() *GlobalCfg {
	endpoints := strings.Split(g.cfg.Etcd.Endpoints, ",")
	e, err := etcd.Connect(endpoints, constants.EtcdPrefix)
	if err != nil {
		logger.Critical(nil, "%+s", "Failed to connect etcd...")
		panic(err)
	}
	logger.Debug(nil, "%+s", "Connect to etcd successfully.")
	g.etcd = e
	logger.Debug(nil, "%+s", "Set globalcfg etcd value.")
	return g
}

func (g *GlobalCfg) setLoggerLevel() *GlobalCfg {
	AppLogMode := config.GetInstance().Log.Level
	logger.SetLevelByString(AppLogMode)
	logger.Debug(nil, "Set app log level to %+s", AppLogMode)
	return g
}

func (g *GlobalCfg) GetEtcd() *etcd.Etcd {
	return g.etcd
}
