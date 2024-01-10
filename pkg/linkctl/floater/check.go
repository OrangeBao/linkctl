package floater

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	ctlutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/kosmos.io/linkctl/pkg/linkctl/floater/command"
	"github.com/kosmos.io/linkctl/pkg/linkctl/floater/netmap"
	"github.com/kosmos.io/linkctl/pkg/linkctl/util"
	"github.com/kosmos.io/linkctl/pkg/utils"
	"github.com/kosmos.io/linkctl/pkg/version"
)

var checkExample = templates.Examples(i18n.T(`
        # Check single cluster network connectivity, e.g:
        linkctl check --src-kubeconfig ~/kubeconfig/src-kubeconfig
        
        # Check across clusters network connectivity, e.g:
        linkctl check --src-kubeconfig ~/kubeconfig/src-kubeconfig --dst-kubeconfig ~/kubeconfig/dst-kubeconfig
        
        # Check cluster network connectivity, if you need to specify a special image repository, e.g: 
        linkctl check -r ghcr.io/kosmos-io
`))

var (
	once sync.Once
)

type CommandCheckOptions struct {
	Namespace          string `json:"namespace,omitempty"`
	ImageRepository    string `json:"imageRepository,omitempty"`
	DstImageRepository string `json:"dstImageRepository,omitempty"`
	Version            string `json:"version,omitempty"`

	Protocol    string `json:"protocol,omitempty"`
	PodWaitTime int    `json:"podWaitTime,omitempty"`
	Port        string `json:"port,omitempty"`
	HostNetwork bool   `json:"hostNetwork,omitempty"`

	SrcKubeConfig string `json:"srcKubeConfig,omitempty"`
	DstKubeConfig string `json:"dstKubeConfig,omitempty"`

	MaxNum int `json:"maxNum,omitempty"`

	AutoClean bool `json:"autoClean,omitempty"`

	CmdTimeout int `json:"cmdTimeout,omitempty"`

	SrcFloater *Floater `json:"-"`
	DstFloater *Floater `json:"-"`

	ResumeRecord []*PrintCheckData `json:"-"`
}

type PrintCheckData struct {
	command.Result
	SrcNodeName string `json:"srcNodeName"`
	DstNodeName string `json:"dstNodeName"`
	TargetIP    string `json:"targetIP"`
}

func NewOptions() (*cobra.Command, *CommandCheckOptions) {
	o := &CommandCheckOptions{
		Version: version.GetReleaseVersion().PatchRelease(),
	}
	cmd := &cobra.Command{
		Use:                   "check",
		Short:                 i18n.T("Check network connectivity between Kosmos clusters"),
		Long:                  "",
		Example:               checkExample,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctlutil.CheckErr(o.Complete())
			ctlutil.CheckErr(o.Validate())
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

	flags := cmd.Flags()
	flags.StringVarP(&o.Namespace, "namespace", "n", utils.DefaultNamespace, "Kosmos namespace.")
	flags.StringVarP(&o.ImageRepository, "image-repository", "r", utils.DefaultImageRepository, "Image repository.")
	flags.StringVarP(&o.DstImageRepository, "dst-image-repository", "", "", "Destination cluster image repository.")
	flags.StringVar(&o.SrcKubeConfig, "src-kubeconfig", "", "Absolute path to the source cluster kubeconfig file.")
	flags.StringVar(&o.DstKubeConfig, "dst-kubeconfig", "", "Absolute path to the destination cluster kubeconfig file.")
	flags.BoolVar(&o.HostNetwork, "host-network", false, "Configure HostNetwork.")
	flags.StringVar(&o.Port, "port", "8889", "Port used by floater.")
	flags.IntVarP(&o.PodWaitTime, "pod-wait-time", "w", 30, "Time for wait pod(floater) launch.")
	flags.StringVar(&o.Protocol, "protocol", string(TCP), "Protocol for the network problem.")
	flags.IntVar(&o.MaxNum, "max-num", 3, "Max number of go-route to lanuch.")
	flags.BoolVar(&o.AutoClean, "auto-clean", false, "Auto clean the pods.")
	flags.IntVar(&o.CmdTimeout, "cmd-timeout", 3, "Timeout for the command.")

	return cmd, o
}

func NewCmdCheck() *cobra.Command {
	cmd, _ := NewOptions()
	return cmd
}

func (o *CommandCheckOptions) LoadConfig() {
	fromConfig := &CommandCheckOptions{}
	if err := util.ReadOpt(fromConfig); err == nil {
		once.Do(func() {
			klog.Infof("use config from file!!!!!!")
		})
		o.Namespace = fromConfig.Namespace
		o.ImageRepository = fromConfig.ImageRepository
		o.DstImageRepository = fromConfig.DstImageRepository
		o.SrcKubeConfig = fromConfig.SrcKubeConfig
		o.DstKubeConfig = fromConfig.DstKubeConfig
		o.HostNetwork = fromConfig.HostNetwork
		o.Port = fromConfig.Port
		o.PodWaitTime = fromConfig.PodWaitTime
		o.Protocol = fromConfig.Protocol
		o.MaxNum = fromConfig.MaxNum
		o.AutoClean = fromConfig.AutoClean
		o.CmdTimeout = fromConfig.CmdTimeout
		o.Version = fromConfig.Version
	}
}

func (o *CommandCheckOptions) Complete() error {
	// load config from config.json
	o.LoadConfig()

	if len(o.DstImageRepository) == 0 {
		o.DstImageRepository = o.ImageRepository
	}

	srcFloater := NewCheckFloater(o, false)
	if err := srcFloater.completeFromKubeConfigPath(o.SrcKubeConfig); err != nil {
		return err
	}
	o.SrcFloater = srcFloater

	if o.DstKubeConfig != "" {
		dstFloater := NewCheckFloater(o, true)
		if err := dstFloater.completeFromKubeConfigPath(o.DstKubeConfig); err != nil {
			return err
		}
		o.DstFloater = dstFloater
	}

	return nil
}

func (o *CommandCheckOptions) Validate() error {
	if len(o.Namespace) == 0 {
		return fmt.Errorf("namespace must be specified")
	}

	return nil
}

func (o *CommandCheckOptions) Clean() error {
	if err := o.SrcFloater.RemoveFloater(); err != nil {
		return err
	}

	if o.DstKubeConfig != "" {
		if err := o.DstFloater.RemoveFloater(); err != nil {
			return err
		}
	}
	return nil
}

func (o *CommandCheckOptions) Run() error {
	var resultData []*PrintCheckData

	if err := o.SrcFloater.CreateFloater(); err != nil {
		return err
	}

	if o.DstKubeConfig != "" {
		if o.DstFloater.EnableHostNetwork {
			srcNodeInfos, err := o.SrcFloater.GetNodesInfo()
			if err != nil {
				return fmt.Errorf("get src cluster nodeInfos failed: %s", err)
			}

			if err = o.DstFloater.CreateFloater(); err != nil {
				return err
			}
			var dstNodeInfos []*FloatInfo
			dstNodeInfos, err = o.DstFloater.GetNodesInfo()
			if err != nil {
				return fmt.Errorf("get dist cluster nodeInfos failed: %s", err)
			}

			resultData = o.RunNative(srcNodeInfos, dstNodeInfos)
		} else {
			srcPodInfos, err := o.SrcFloater.GetPodInfo()
			if err != nil {
				return fmt.Errorf("get src cluster podInfos failed: %s", err)
			}

			if err = o.DstFloater.CreateFloater(); err != nil {
				return err
			}
			var dstPodInfos []*FloatInfo
			dstPodInfos, err = o.DstFloater.GetPodInfo()
			if err != nil {
				return fmt.Errorf("get dist cluster podInfos failed: %s", err)
			}

			resultData = o.RunRange(srcPodInfos, dstPodInfos)
		}
	} else {
		if o.SrcFloater.EnableHostNetwork {
			srcNodeInfos, err := o.SrcFloater.GetNodesInfo()
			if err != nil {
				return fmt.Errorf("get src cluster nodeInfos failed: %s", err)
			}
			resultData = o.RunNative(srcNodeInfos, srcNodeInfos)
		} else {
			srcPodInfos, err := o.SrcFloater.GetPodInfo()
			if err != nil {
				return fmt.Errorf("get src cluster podInfos failed: %s", err)
			}
			resultData = o.RunRange(srcPodInfos, srcPodInfos)
		}
	}

	o.PrintResult(resultData)

	if o.AutoClean {
		if err := o.Clean(); err != nil {
			return err
		}
	}

	// save options for resume
	o.SaveOpts()

	return nil
}

func (o *CommandCheckOptions) SaveOpts() {
	if err := util.WriteOpt(o); err != nil {
		klog.Fatal(err)
	} else {
		klog.Info("write opts success")
	}
}

func (o *CommandCheckOptions) Skip(podInfo *FloatInfo, targetIP string) bool {
	// is check:  no skip
	if len(o.ResumeRecord) == 0 {
		return false
	}
	// is resume: filt
	for _, r := range o.ResumeRecord {
		if r.SrcNodeName == podInfo.NodeName && r.TargetIP == targetIP {
			return false
		}
	}
	return true
}

func (o *CommandCheckOptions) RunRange(iPodInfos []*FloatInfo, jPodInfos []*FloatInfo) []*PrintCheckData {
	var resultData []*PrintCheckData
	mutex := sync.Mutex{}

	barctl := utils.NewBar(len(jPodInfos) * len(iPodInfos))

	worker := func(iPodInfo *FloatInfo) {
		for _, jPodInfo := range jPodInfos {
			for _, ip := range jPodInfo.PodIPs {
				var targetIP string
				var err error
				var cmdResult *command.Result
				if o.DstFloater != nil {
					targetIP, err = netmap.NetMap(ip, o.DstFloater.CIDRsMap)
				} else {
					targetIP = ip
				}
				if err != nil {
					cmdResult = command.ParseError(err)
				} else {
					// isSkip
					if o.Skip(iPodInfo, targetIP) {
						continue
					}
					// ToDo RunRange && RunNative func support multiple commands, and the code needs to be optimized
					cmdObj := &command.Ping{
						TargetIP: targetIP,
					}
					cmdResult = o.SrcFloater.CommandExec(iPodInfo, cmdObj)
				}
				mutex.Lock()
				resultData = append(resultData, &PrintCheckData{
					*cmdResult,
					iPodInfo.NodeName, jPodInfo.NodeName, targetIP,
				})
				mutex.Unlock()
			}
			barctl.Add(1)
		}
	}

	var wg sync.WaitGroup
	ch := make(chan struct{}, o.MaxNum)

	if len(iPodInfos) > 0 && len(jPodInfos) > 0 {
		for _, iPodInfo := range iPodInfos {
			podInfo := iPodInfo
			ch <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				worker(podInfo)
				<-ch
			}()
		}
	}

	wg.Wait()

	return resultData
}

func (o *CommandCheckOptions) RunNative(iNodeInfos []*FloatInfo, jNodeInfos []*FloatInfo) []*PrintCheckData {
	var resultData []*PrintCheckData

	barctl := utils.NewBar(len(iNodeInfos) * len(jNodeInfos))

	worker := func(iNodeInfo *FloatInfo) {
		for _, jNodeInfo := range jNodeInfos {
			for _, ip := range jNodeInfo.NodeIPs {
				// isSkip
				if o.Skip(iNodeInfo, ip) {
					continue
				}
				// ToDo RunRange && RunNative func support multiple commands, and the code needs to be optimized
				cmdObj := &command.Ping{
					TargetIP: ip,
				}
				cmdResult := o.SrcFloater.CommandExec(iNodeInfo, cmdObj)
				resultData = append(resultData, &PrintCheckData{
					*cmdResult,
					iNodeInfo.NodeName, jNodeInfo.NodeName, ip,
				})
			}
			barctl.Add(1)
		}
	}

	var wg sync.WaitGroup
	ch := make(chan struct{}, o.MaxNum)

	if len(iNodeInfos) > 0 && len(jNodeInfos) > 0 {
		for _, iNodeInfo := range iNodeInfos {
			nodeInfo := iNodeInfo
			ch <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				worker(nodeInfo)
				<-ch
			}()
		}
	}

	wg.Wait()

	return resultData
}

func (o *CommandCheckOptions) PrintResult(resultData []*PrintCheckData) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"S/N", "SRC_NODE_NAME", "DST_NODE_NAME", "TARGET_IP", "RESULT"})

	tableException := tablewriter.NewWriter(os.Stdout)
	tableException.SetHeader([]string{"S/N", "SRC_NODE_NAME", "DST_NODE_NAME", "TARGET_IP", "RESULT", "LOG"})

	tableFailed := tablewriter.NewWriter(os.Stdout)
	tableFailed.SetHeader([]string{"S/N", "SRC_NODE_NAME", "DST_NODE_NAME", "TARGET_IP", "RESULT", "LOG"})

	resumeData := []*PrintCheckData{}

	for index, r := range resultData {
		// klog.Infof(fmt.Sprintf("%s %s %v", r.SrcNodeName, r.DstNodeName, r.IsSucceed))
		row := []string{strconv.Itoa(index + 1), r.SrcNodeName, r.DstNodeName, r.TargetIP, command.PrintStatus(r.Status), r.ResultStr}
		if r.Status == command.CommandFailed {
			resumeData = append(resumeData, r)
			tableFailed.Rich(row, []tablewriter.Colors{
				{},
				{tablewriter.Bold, tablewriter.FgHiRedColor},
				{tablewriter.Bold, tablewriter.FgHiRedColor},
				{tablewriter.Bold, tablewriter.FgHiRedColor},
				{tablewriter.Bold, tablewriter.FgHiRedColor},
			})
		} else if r.Status == command.ExecError {
			resumeData = append(resumeData, r)
			tableException.Rich(row, []tablewriter.Colors{
				{},
				{tablewriter.Bold, tablewriter.FgCyanColor},
				{tablewriter.Bold, tablewriter.FgCyanColor},
				{tablewriter.Bold, tablewriter.FgCyanColor},
				{tablewriter.Bold, tablewriter.FgCyanColor},
			})
		} else {
			// resumeData = append(resumeData, r)
			table.Rich(row[:len(row)-1], []tablewriter.Colors{
				{},
				{tablewriter.Bold, tablewriter.FgGreenColor},
				{tablewriter.Bold, tablewriter.FgGreenColor},
				{tablewriter.Bold, tablewriter.FgGreenColor},
				{tablewriter.Bold, tablewriter.FgGreenColor},
			})
		}
	}
	fmt.Println("")
	table.Render()
	fmt.Println("")
	tableException.Render()

	util.WriteResume(resumeData)
}
