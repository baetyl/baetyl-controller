package kube

import (
	"context"

	"gopkg.in/tomb.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	clientSet "github.com/baetyl/baetyl-controller/kube/client/clientset/versioned"
	"github.com/baetyl/baetyl-controller/kube/signals"
	"github.com/baetyl/baetyl-controller/utils"
)

const (
	DefaultThreadiness = 2
)

type Controller interface {
	Start()
	Run() error
}

type Client struct {
	ctx context.Context

	kubeClient   kubernetes.Interface
	customClient clientSet.Interface

	controllers map[string]Controller
	tomb.Tomb
}

func NewController(ctx context.Context, path string) (*Client, error) {
	var cfg Config
	if err := utils.LoadYAML(path, &cfg); err != nil {
		return nil, err
	}

	kubeConfig, err := func() (*rest.Config, error) {
		if !cfg.Kube.OutCluster {
			return rest.InClusterConfig()
		}
		return clientcmd.BuildConfigFromFlags(
			"", cfg.Kube.ConfigPath)
	}()
	if err != nil {
		return nil, err
	}

	// client
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	customClient, err := clientSet.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	// TODO optimization
	controllers := map[string]Controller{}
	tpl, err := NewTemplateClient(ctx, kubeClient, customClient, DefaultThreadiness, signals.SetupSignalHandler())
	if err != nil {
		return nil, err
	}
	controllers["template"] = tpl

	return &Client{
		ctx:          ctx,
		kubeClient:   kubeClient,
		customClient: customClient,
		controllers:  controllers,
	}, nil
}

func (c *Client) Run() {
	for k, v := range c.controllers {
		klog.Info("load controller", k)
		v.Start()
		c.Tomb.Go(v.Run)
	}
}

func (c *Client) Close() error {
	c.Tomb.Kill(nil)
	return c.Wait()
}
