//go:generate go run pkg/codegen/cleanup/main.go
//go:generate /bin/rm -rf pkg/generated
//go:generate go run pkg/codegen/main.go

package main

import (
	"context"
	"os"

	"github.com/rancher/wrangler/pkg/signals"
	"github.com/urfave/cli"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/rancher/harvester-network-controller/pkg/config"
	"github.com/rancher/harvester-network-controller/pkg/controller/daemonset"
)

var (
	VERSION = "v0.0.0-dev"
)

func main() {
	app := cli.NewApp()
	app.Name = "harvester-network-controller"
	app.Version = VERSION
	app.Usage = "Harvester Network Controller, to help with cluster network configuration. Options kubeconfig or masterurl are required."
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "kubeconfig, k",
			EnvVar: "KUBECONFIG",
			Value:  "",
			Usage:  "Kubernetes config files, e.g. $HOME/.kube/config",
		},
		cli.StringFlag{
			Name:   "master, m",
			EnvVar: "MASTERURL",
			Value:  "",
			Usage:  "Kubernetes cluster master URL.",
		},
		cli.StringFlag{
			Name:   "namespace, n",
			EnvVar: "NAMESPACE",
			Value:  "",
			Usage:  "Namespace to watch, empty means it will watch CRDs in all namespaces.",
		},
		cli.IntFlag{
			Name:   "threads, t",
			EnvVar: "THREADS",
			Value:  2,
			Usage:  "Threadiness level to set, defaults to 2.",
		},
	}
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		klog.Fatal(err)
	}
}

func run(c *cli.Context) error {
	masterURL := c.String("master")
	kubeconfig := c.String("kubeconfig")
	namespace := c.String("namespace")
	threadiness := c.Int("threads")

	if threadiness <= 0 {
		klog.Infof("Can not start with thread count of %d, please pass a proper thread count.", threadiness)
		return nil
	}

	klog.Infof("Starting network controller with %d threads.", threadiness)

	if namespace == "" {
		klog.Info("Starting network controller with no namespace.")
	} else {
		klog.Infof("Starting network controller in namespace: %s.", namespace)
	}

	ctx := signals.SetupSignalHandler(context.Background())

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building config from flags: %s", err.Error())
	}

	management, err := config.SetupManagement(ctx, cfg)
	if err != nil {
		klog.Fatalf("Error building harvester controllers: %s", err.Error())
	}

	if err := management.Register(ctx, daemonset.RegisterFuncList); err != nil {
		klog.Fatalf("Error Register: %s", err.Error())
	}

	if err := management.Start(threadiness); err != nil {
		klog.Fatalf("Error starting: %s", err.Error())
	}

	<-ctx.Done()
	return nil
}
