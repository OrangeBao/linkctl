package floater

import (
	"github.com/spf13/cobra"
	ctlutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/kosmos.io/linkctl/pkg/linkctl/util"
)

type CommandResumeOptions struct {
	CommandCheckOptions
}

func NewCmdResume() *cobra.Command {
	cmd, checkOpt := NewOptions()

	o := &CommandResumeOptions{}
	o.CommandCheckOptions = *checkOpt
	cmd.Use = "resume"
	cmd.Short = i18n.T("resume network connectivity between Kosmos clusters")
	cmd.Example = checkExample
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctlutil.CheckErr(o.Complete())
		ctlutil.CheckErr(o.Validate())
		ctlutil.CheckErr(o.Run())
		return nil
	}
	return cmd
}

func (o *CommandResumeOptions) Run() error {
	var resumeData []*PrintCheckData

	util.ReadResume(&resumeData)

	o.CommandCheckOptions.ResumeRecord = resumeData

	return o.CommandCheckOptions.Run()
}
