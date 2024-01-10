package floater

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"

	"github.com/kosmos.io/linkctl/pkg/linkctl/floater/command"
	"github.com/kosmos.io/linkctl/pkg/linkctl/manifest"
	"github.com/kosmos.io/linkctl/pkg/linkctl/util"
	"github.com/kosmos.io/linkctl/pkg/utils"
)

type Protocol string

const (
	TCP  Protocol = "tcp"
	UDP  Protocol = "udp"
	IPv4 Protocol = "ipv4"
)

const (
	DefaultFloaterName = "clusterlink-floater"
)

type FloatInfo struct {
	NodeName string
	NodeIPs  []string

	PodName string
	PodIPs  []string
}

func (i *FloatInfo) String() string {
	return fmt.Sprintf("nodeName: %s, nodeIPs: %s, podName: %s, podIPs: %s", i.NodeName, i.NodeIPs, i.PodName, i.PodIPs)
}

type Floater struct {
	Namespace         string
	Name              string
	ImageRepository   string
	Version           string
	PodWaitTime       int
	Port              string
	EnableHostNetwork bool
	EnableAnalysis    bool

	CIDRsMap map[string]string

	Config *rest.Config
	Client kubernetes.Interface

	CmdTimeout int
}

func NewCheckFloater(o *CommandCheckOptions, isDst bool) *Floater {
	imageRepository := o.ImageRepository
	if isDst {
		imageRepository = o.DstImageRepository
	}
	floater := &Floater{
		Namespace:         o.Namespace,
		Name:              DefaultFloaterName,
		ImageRepository:   imageRepository,
		Version:           o.Version,
		PodWaitTime:       o.PodWaitTime,
		Port:              o.Port,
		EnableHostNetwork: false,
		EnableAnalysis:    false,
		CmdTimeout:        o.CmdTimeout,
	}
	if o.HostNetwork {
		floater.EnableHostNetwork = true
	}
	return floater
}

func (f *Floater) completeFromKubeConfigPath(kubeConfigPath string) error {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return fmt.Errorf("linkctl docter complete error, generate floater config failed: %v", err)
	}
	f.Config = config

	f.Client, err = kubernetes.NewForConfig(f.Config)
	if err != nil {
		return fmt.Errorf("linkctl docter complete error, generate floater client failed: %v", err)
	}

	return nil
}

func (f *Floater) CreateFloater() error {
	klog.Infof("create Clusterlink floater, namespace: %s", f.Namespace)
	namespace := &corev1.Namespace{}
	namespace.Name = f.Namespace
	_, err := f.Client.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("linkctl floater run error, namespace options failed: %v", err)
		}
	}

	klog.Info("create Clusterlink floater, apply RBAC")
	if err = f.applyServiceAccount(); err != nil {
		return err
	}
	if err = f.applyClusterRole(); err != nil {
		return err
	}
	if err = f.applyClusterRoleBinding(); err != nil {
		return err
	}

	klog.Infof("create Clusterlink floater, version: %s", f.Version)
	if err = f.applyDaemonSet(); err != nil {
		return err
	}

	return nil
}

func (f *Floater) applyServiceAccount() error {
	clusterlinkFloaterServiceAccount, err := util.GenerateServiceAccount(manifest.ClusterlinkFloaterServiceAccount, manifest.ServiceAccountReplace{
		Namespace: f.Namespace,
	})
	if err != nil {
		return err
	}
	_, err = f.Client.CoreV1().ServiceAccounts(f.Namespace).Create(context.TODO(), clusterlinkFloaterServiceAccount, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("linkctl floater run error, serviceaccount options failed: %v", err)
		}
	}

	return nil
}

func (f *Floater) applyClusterRole() error {
	clusterlinkFloaterClusterRole, err := util.GenerateClusterRole(manifest.ClusterlinkFloaterClusterRole, nil)
	if err != nil {
		return err
	}
	_, err = f.Client.RbacV1().ClusterRoles().Create(context.TODO(), clusterlinkFloaterClusterRole, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("linkctl floater run error, clusterrole options failed: %v", err)
		}
	}

	return nil
}

func (f *Floater) applyClusterRoleBinding() error {
	clusterlinkFloaterClusterRoleBinding, err := util.GenerateClusterRoleBinding(manifest.ClusterlinkFloaterClusterRoleBinding, manifest.ClusterRoleBindingReplace{
		Namespace: f.Namespace,
	})
	if err != nil {
		return err
	}
	_, err = f.Client.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterlinkFloaterClusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("linkctl floater run error, clusterrolebinding options failed: %v", err)
		}
	}

	return nil
}

func (f *Floater) applyDaemonSet() error {
	clusterlinkFloaterDaemonSet, err := util.GenerateDaemonSet(manifest.ClusterlinkFloaterDaemonSet, manifest.DaemonSetReplace{
		Namespace:         f.Namespace,
		Name:              f.Name,
		Version:           f.Version,
		ImageRepository:   f.ImageRepository,
		Port:              f.Port,
		EnableHostNetwork: f.EnableHostNetwork,
		EnableAnalysis:    f.EnableAnalysis,
	})
	if err != nil {
		return err
	}
	_, err = f.Client.AppsV1().DaemonSets(f.Namespace).Create(context.Background(), clusterlinkFloaterDaemonSet, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("linkctl floater run error, daemonset options failed: %v", err)
		}
	}

	floaterLabel := map[string]string{"app": f.Name}
	if err = util.WaitPodReady(f.Client, f.Namespace, util.MapToString(floaterLabel), f.PodWaitTime); err != nil {
		klog.Warningf("exist cluster node startup floater timeout, error: %v", err)
	}

	return nil
}

func (f *Floater) GetPodInfo() ([]*FloatInfo, error) {
	selector := util.MapToString(map[string]string{"app": f.Name})
	pods, err := f.Client.CoreV1().Pods(f.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pods in %s with selector %s", f.Namespace, selector)
	}

	var floaterInfos []*FloatInfo
	for _, pod := range pods.Items {
		podInfo := &FloatInfo{
			NodeName: pod.Spec.NodeName,
			PodName:  pod.GetObjectMeta().GetName(),
			PodIPs:   podIPToArray(pod.Status.PodIPs),
		}

		floaterInfos = append(floaterInfos, podInfo)
	}

	return floaterInfos, nil
}

func podIPToArray(podIPs []corev1.PodIP) []string {
	var ret []string

	for _, podIP := range podIPs {
		ret = append(ret, podIP.IP)
	}

	return ret
}

func (f *Floater) GetNodesInfo() ([]*FloatInfo, error) {
	selector := util.MapToString(map[string]string{"app": f.Name})
	pods, err := f.Client.CoreV1().Pods(f.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pods in %s with selector %s", f.Namespace, selector)
	}

	nodes, err := f.Client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(nodes.Items) == 0 {
		return nil, fmt.Errorf("unable to list any node")
	}

	var floaterInfos []*FloatInfo
	for _, pod := range pods.Items {
		for _, node := range nodes.Items {
			if pod.Spec.NodeName == node.Name {
				nodeInfo := &FloatInfo{
					NodeName: node.Name,
					NodeIPs:  nodeIPToArray(node),
					PodName:  pod.Name,
				}
				floaterInfos = append(floaterInfos, nodeInfo)
			}
		}
	}

	return floaterInfos, nil
}

func nodeIPToArray(node corev1.Node) []string {
	var nodeIPs []string

	for _, addr := range node.Status.Addresses {
		if addr.Type == "InternalIP" {
			nodeIPs = append(nodeIPs, addr.Address)
		}
	}

	return nodeIPs
}

func (f *Floater) GetCmdTimeout() time.Duration {
	if f.CmdTimeout == 0 {
		return 3 * time.Second
	} else {
		return time.Duration(f.CmdTimeout) * time.Second
	}
}

func (f *Floater) CommandExec(fInfo *FloatInfo, cmd command.Command) *command.Result {
	req := f.Client.CoreV1().RESTClient().Post().Resource("pods").Namespace(f.Namespace).Name(fInfo.PodName).
		SubResource("exec").
		Param("container", "floater").
		Param("command", "/bin/sh").
		Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "true").
		Param("tty", "false")

	outBuffer := &bytes.Buffer{}
	errBuffer := &bytes.Buffer{}

	exec, err := remotecommand.NewSPDYExecutor(f.Config, "POST", req.URL())
	if err != nil {
		return command.ParseError(err)
	}

	// timeout 5s
	ctx, cancel := context.WithTimeout(context.Background(), f.GetCmdTimeout())
	defer cancel()
	cmdStr := cmd.GetCommandStr()

	// klog.Infof("cmdStr: %s", cmdStr)
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  strings.NewReader(cmdStr),
		Stdout: outBuffer,
		Stderr: errBuffer,
		Tty:    false,
	})

	if err != nil {
		// klog.Infof("error: %s", err)
		return command.ParseError(fmt.Errorf("%s, stderr: %s", err, errBuffer.String()))
	}

	return cmd.ParseResult(outBuffer.String())
}

func (f *Floater) RemoveFloater() error {
	klog.Infof("remove Clusterlink floater, version: %s", f.Version)
	if err := f.removeDaemonSet(); err != nil {
		return err
	}

	klog.Info("remove Clusterlink floater, apply RBAC")
	if err := f.removeClusterRoleBinding(); err != nil {
		return err
	}
	if err := f.removeClusterRole(); err != nil {
		return err
	}
	if err := f.removeServiceAccount(); err != nil {
		return err
	}

	if f.Namespace != utils.DefaultNamespace {
		klog.Infof("remove namespace specified when creating Clusterlink floater, namespace: %s", f.Namespace)
		err := f.Client.CoreV1().Namespaces().Delete(context.TODO(), f.Namespace, metav1.DeleteOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("linkctl floater run error, namespace options failed: %v", err)
			}
		}
	}

	return nil
}

func (f *Floater) removeDaemonSet() error {
	err := f.Client.AppsV1().DaemonSets(f.Namespace).Delete(context.Background(), f.Name, metav1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("linkctl floater run error, daemonset options failed: %v", err)
		}
	}

	return nil
}

func (f *Floater) removeClusterRoleBinding() error {
	err := f.Client.RbacV1().ClusterRoleBindings().Delete(context.Background(), f.Name, metav1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("linkctl floater run error, clusterrolebinding options failed: %v", err)
		}
	}

	return nil
}

func (f *Floater) removeClusterRole() error {
	err := f.Client.RbacV1().ClusterRoles().Delete(context.Background(), f.Name, metav1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("linkctl floater run error, clusterrole options failed: %v", err)
		}
	}

	return nil
}

func (f *Floater) removeServiceAccount() error {
	err := f.Client.CoreV1().ServiceAccounts(f.Namespace).Delete(context.Background(), f.Name, metav1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("linkctl floater run error, serviceaccount options failed: %v", err)
		}
	}

	return nil
}
