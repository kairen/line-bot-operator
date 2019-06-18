package main

import (
	goflag "flag"
	"fmt"
	"os"

	"github.com/kairen/line-bot-operator/pkg/operator"
	"github.com/kairen/line-bot-operator/pkg/version"
	flag "github.com/spf13/pflag"
	"k8s.io/klog"
)

var (
	kubeconfig string
	ver        bool
)

func parserFlags() {
	flag.StringVarP(&kubeconfig, "kubeconfig", "", "", "Absolute path to the kubeconfig file.")
	flag.BoolVarP(&ver, "version", "", false, "Display the version.")
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
}

func main() {
	klog.InitFlags(nil)
	parserFlags()

	klog.Infof("Starting LINE bot operator...")

	if ver {
		fmt.Fprintf(os.Stdout, "%s\n", version.GetVersion())
		os.Exit(0)
	}

	op := operator.NewMainOperator()
	if err := op.Initialize(kubeconfig); err != nil {
		klog.Fatalf("Error initing operator instance: %+v.", err)
	}

	if err := op.Run(); err != nil {
		klog.Fatalf("Error serving operator instance: %+v.", err)
	}
}
