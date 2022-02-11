package main

import (
	"context"
	"os"

	"k8s.io/klog/v2"

	"github.com/baetyl/baetyl-controller/api"
	"github.com/baetyl/baetyl-controller/kube"
	"github.com/baetyl/baetyl-controller/server"
)

func main() {
	err := run()
	if err != nil {
		klog.Fatal(err.Error())
		os.Exit(-1)
	}
}

func run() error {
	klog.InitFlags(nil)

	cli, err := kube.NewController(context.Background(), "etc/config.yaml")
	if err != nil {
		return err
	}
	cli.Run()

	a, err := api.NewAdminAPI(cli)
	if err != nil {
		return err
	}

	svr, err := server.NewServer("etc/config.yaml")
	if err != nil {
		return err
	}
	svr.SetAPI(a)
	go svr.Run()
	defer svr.Close()

	var sig chan struct{}
	<-sig
	cli.Close()

	return nil
}
