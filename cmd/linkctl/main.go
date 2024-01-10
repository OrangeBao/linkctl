package main

import (
	"k8s.io/component-base/cli"
	"k8s.io/kubectl/pkg/cmd/util"

	app "github.com/kosmos.io/linkctl/pkg/linkctl"
)

func main() {
	cmd := app.NewKosmosCtlCommand()
	if err := cli.RunNoErrOutput(cmd); err != nil {
		util.CheckErr(err)
	}
}
