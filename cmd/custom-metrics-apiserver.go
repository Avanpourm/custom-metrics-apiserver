/*
Description: custom indicator collection interface
By JamesBryce Avanpourm
Create all: JamesBryce Avanpourm
Last modified: 2019-01-11
*/

package main

import (
	"flag"
	"os"
	"runtime"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/cmd/app"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	cmd := app.NewCommandStartCustomMetricsServer(os.Stdout, os.Stderr, wait.NeverStop)

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
