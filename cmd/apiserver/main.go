// Copyright 2018 The OpenPitrix Authors. All rights reserved.
// Use of this source code is governed by a Apache license
// that can be found in the LICENSE file.

package main

import (
	"os"
	"os/signal"
	"syscall"

	"openpitrix.io/scheduler/pkg/config"
	"openpitrix.io/scheduler/pkg/services/apiserver"
)

func exitHandler() {
	c := make(chan os.Signal)

	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for s := range c {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				ExitFunc()
			case syscall.SIGUSR1:
			case syscall.SIGUSR2:
			default:
			}
		}
	}()
}

func ExitFunc() {
	os.Exit(0)
}

func main() {
	exitHandler()

	config.GetInstance().LoadConf()

	apiserver.Run()
}
