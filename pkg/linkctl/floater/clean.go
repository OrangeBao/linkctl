package floater

import (
	"github.com/spf13/cobra"
	ctlutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
)

type CommandCleanOptions struct {
	CommandCheckOptions
}

func NewCmdClean() *cobra.Command {
	cmd, checkOpt := NewOptions()

	o := &CommandResumeOptions{}
	o.CommandCheckOptions = *checkOpt
	cmd.Use = "clean"
	cmd.Short = i18n.T("clean network connectivity between Kosmos clusters")
	cmd.Example = checkExample
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctlutil.CheckErr(o.Complete())
		ctlutil.CheckErr(o.Validate())
		ctlutil.CheckErr(o.Run())
		return nil
	}
	return cmd
}

func (o *CommandCleanOptions) Run() error {
	return o.Clean()
}
