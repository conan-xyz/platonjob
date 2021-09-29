package main

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/yaml.v2"
	"k8s.io/klog"

	"gitee.com/zonzpoo/platonjob/conf"
	"gitee.com/zonzpoo/platonjob/sched"
)

var (
	confPath string
	cmd      string
	ac       *conf.Config
)

func init() {
	ac = new(conf.Config)

	// flag init.
	flag.StringVar(&confPath, "config", "config/config.yaml", "c config file path")
	flag.StringVar(&cmd, "cmd", "none", "exec command")
}

func loadConf(path string) error {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlFile, ac)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()

	err := loadConf(confPath)
	if err != nil {
		panic(err)
	}

	klog.InitFlags(nil)

	c := sched.NewController(context.Background(), ac)

	go func() {
		term := make(chan os.Signal)
		signal.Notify(term, os.Interrupt, syscall.SIGTERM)
		select {
		case <-term:
			klog.Info("Received SIGTERM, try exiting gracefully...")
			if err := c.Stop(); err != nil {
				klog.Infof("Error during shutdown: %v", err)
			}
		}
	}()

	c.Start()
}
