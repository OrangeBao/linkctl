package util

import (
	"bytes"
	"fmt"
	"text/template"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	ClusterGVR     = schema.GroupVersionResource{Group: "kosmos.io", Version: "v1alpha1", Resource: "clusters"}
	ClusterNodeGVR = schema.GroupVersionResource{Group: "kosmos.io", Version: "v1alpha1", Resource: "clusternodes"}
	NodeConfigGVR  = schema.GroupVersionResource{Group: "kosmos.io", Version: "v1alpha1", Resource: "nodeconfigs"}
)

func GenerateDeployment(deployTemplate string, obj interface{}) (*appsv1.Deployment, error) {
	deployBytes, err := parseTemplate(deployTemplate, obj)
	if err != nil {
		return nil, fmt.Errorf("linkctl parsing Deployment template exception, error: %v", err)
	} else if deployBytes == nil {
		return nil, fmt.Errorf("linkctl get Deployment template exception, value is empty")
	}

	deployStruct := &appsv1.Deployment{}

	if err = runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), deployBytes, deployStruct); err != nil {
		return nil, fmt.Errorf("linkctl decode deployBytes error: %v", err)
	}

	return deployStruct, nil
}

func GenerateDaemonSet(dsTemplate string, obj interface{}) (*appsv1.DaemonSet, error) {
	dsBytes, err := parseTemplate(dsTemplate, obj)
	if err != nil {
		return nil, fmt.Errorf("linkctl parsing DaemonSet template exception, error: %v", err)
	} else if dsBytes == nil {
		return nil, fmt.Errorf("linkctl get DaemonSet template exception, value is empty")
	}

	dsStruct := &appsv1.DaemonSet{}

	if err = runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), dsBytes, dsStruct); err != nil {
		return nil, fmt.Errorf("linkctl decode dsBytes error: %v", err)
	}

	return dsStruct, nil
}

func GenerateServiceAccount(saTemplate string, obj interface{}) (*corev1.ServiceAccount, error) {
	saBytes, err := parseTemplate(saTemplate, obj)
	if err != nil {
		return nil, fmt.Errorf("linkctl parsing ServiceAccount template exception, error: %v", err)
	} else if saBytes == nil {
		return nil, fmt.Errorf("linkctl get ServiceAccount template exception, value is empty")
	}

	saStruct := &corev1.ServiceAccount{}

	if err = runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), saBytes, saStruct); err != nil {
		return nil, fmt.Errorf("linkctl decode saBytes error: %v", err)
	}

	return saStruct, nil
}

func GenerateClusterRole(crTemplate string, obj interface{}) (*rbacv1.ClusterRole, error) {
	crBytes, err := parseTemplate(crTemplate, obj)
	if err != nil {
		return nil, fmt.Errorf("linkctl parsing ClusterRole template exception, error: %v", err)
	} else if crBytes == nil {
		return nil, fmt.Errorf("linkctl get ClusterRole template exception, value is empty")
	}

	crStruct := &rbacv1.ClusterRole{}

	if err = runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), crBytes, crStruct); err != nil {
		return nil, fmt.Errorf("linkctl decode crBytes error: %v", err)
	}

	return crStruct, nil
}

func GenerateClusterRoleBinding(crbTemplate string, obj interface{}) (*rbacv1.ClusterRoleBinding, error) {
	crbBytes, err := parseTemplate(crbTemplate, obj)
	if err != nil {
		return nil, fmt.Errorf("linkctl parsing ClusterRoleBinding template exception, error: %v", err)
	} else if crbBytes == nil {
		return nil, fmt.Errorf("linkctl get ClusterRoleBinding template exception, value is empty")
	}

	crbStruct := &rbacv1.ClusterRoleBinding{}

	if err = runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), crbBytes, crbStruct); err != nil {
		return nil, fmt.Errorf("linkctl decode crbBytes error: %v", err)
	}

	return crbStruct, nil
}

func GenerateCustomResourceDefinition(crdTemplate string, obj interface{}) (*apiextensionsv1.CustomResourceDefinition, error) {
	crdBytes, err := parseTemplate(crdTemplate, obj)
	if err != nil {
		return nil, fmt.Errorf("linkctl parsing CustomResourceDefinition template exception, error: %v", err)
	} else if crdBytes == nil {
		return nil, fmt.Errorf("linkctl get CustomResourceDefinition template exception, value is empty")
	}

	crdStruct := &apiextensionsv1.CustomResourceDefinition{}

	if err = runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), crdBytes, crdStruct); err != nil {
		return nil, fmt.Errorf("linkctl decode crdBytes error: %v", err)
	}

	return crdStruct, nil
}

func GenerateCustomResource(crdTemplate string, obj interface{}) ([]byte, error) {
	crBytes, err := parseTemplate(crdTemplate, obj)
	if err != nil {
		return nil, fmt.Errorf("linkctl parsing CustomResource template exception, error: %v", err)
	} else if crBytes == nil {
		return nil, fmt.Errorf("linkctl get CustomResource template exception, value is empty")
	}

	return crBytes, nil
}

func parseTemplate(strTmpl string, obj interface{}) ([]byte, error) {
	var buf bytes.Buffer
	tmpl, err := template.New("template").Parse(strTmpl)
	if err != nil {
		return nil, fmt.Errorf("error when parsing template: %v", err)
	}
	err = tmpl.Execute(&buf, obj)
	if err != nil {
		return nil, fmt.Errorf("error when executing template: %v", err)
	}
	return buf.Bytes(), nil
}

func GenerateConfigMap(template string, obj interface{}) (*corev1.ConfigMap, error) {
	bs, err := parseTemplate(template, obj)
	if err != nil {
		return nil, fmt.Errorf("linkctl parsing configmap template exception, error: %v", err)
	} else if bs == nil {
		return nil, fmt.Errorf("linkctl get configmap template exception, value is empty")
	}

	o := &corev1.ConfigMap{}

	if err = runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), bs, o); err != nil {
		return nil, fmt.Errorf("linkctl decode configmap bytes error: %v", err)
	}

	return o, nil
}

func GenerateService(template string, obj interface{}) (*corev1.Service, error) {
	bs, err := parseTemplate(template, obj)
	if err != nil {
		return nil, fmt.Errorf("linkctl parsing service template exception, error: %v", err)
	} else if bs == nil {
		return nil, fmt.Errorf("linkctl get service template exception, value is empty")
	}

	o := &corev1.Service{}

	if err = runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), bs, o); err != nil {
		return nil, fmt.Errorf("linkctl decode service bytes error: %v", err)
	}
	return o, nil
}

func GenerateSecret(template string, obj interface{}) (*corev1.Secret, error) {
	bs, err := parseTemplate(template, obj)
	if err != nil {
		return nil, fmt.Errorf("linkctl parsing secret template exception, error: %v", err)
	} else if bs == nil {
		return nil, fmt.Errorf("linkctl get secret template exception, value is empty")
	}

	o := &corev1.Secret{}

	if err = runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), bs, o); err != nil {
		return nil, fmt.Errorf("linkctl decode secret bytes error: %v", err)
	}

	return o, nil
}
