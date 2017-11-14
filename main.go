package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	controller "github.com/rancher/cluster-controller/controller"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/sync/errgroup"
)

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config",
			Usage:  "Kube config for accessing kubernetes cluster",
			EnvVar: "KUBECONFIG",
		},
	}

	app.Action = func(c *cli.Context) error {
		return runControllers(c.String("config"))
	}
	app.Run(os.Args)
}

func runControllers(config string) error {
	logrus.Info("Staring cluster manager")
	ctx, cancel := context.WithCancel(context.Background())
	wg, ctx := errgroup.WithContext(ctx)

	logrus.Info("Staring controllers")
	for name := range controller.GetControllers() {
		logrus.Infof("Starting [%s] controller", name)
		c := controller.GetControllers()[name]
		err := c.Init(config)
		if err != nil {
			return err
		}
		wg.Go(func() error { return c.Run(ctx) })
	}

	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	select {
	case <-term:
		logrus.Infof("Received SIGTERM, shutting down")
	case <-ctx.Done():
	}

	cancel()
	for name, c := range controller.GetControllers() {
		logrus.Infof("Shutting down [%s] controller", name)
		c.Shutdown()
	}

	if err := wg.Wait(); err != nil {
		logrus.Errorf("Unhandled error received, shutting down: [%v]", err)
		os.Exit(1)
	}
	os.Exit(0)
	return nil
}
