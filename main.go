package main

import (
	"context"
	"os"

	"k8s.io/klog/v2"

	"github.com/baetyl/baetyl-controller/kube"
)

func main() {
	klog.InitFlags(nil)

	cli, err := kube.NewController(context.Background(), "etc/config.yaml")
	if err != nil {
		klog.Fatal(err.Error())
		os.Exit(-1)
	}
	cli.Run()

	var sig chan struct{}
	<-sig
	cli.Close()
}
