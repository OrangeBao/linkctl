package floater

import (
	"fmt"

	"github.com/kosmos.io/linkctl/pkg/linkctl/util"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	ctlutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
)

type CommandInitOptions struct{}

func NewCmdInit() *cobra.Command {
	o := &CommandInitOptions{}

	cmd := &cobra.Command{
		Use:                   "init",
		Short:                 i18n.T("init options"),
		Long:                  "",
		Example:               checkExample,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctlutil.CheckErr(o.Run())
			return nil
		},
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), args)
				}
			}
			return nil
		},
	}
	return cmd
}

func (o *CommandInitOptions) Run() error {
	_, opts := NewOptions()
	if err := util.WriteOpt(opts); err != nil {
		klog.Fatal(err)
	} else {
		klog.Info("write opts success")
	}
	return nil
}
