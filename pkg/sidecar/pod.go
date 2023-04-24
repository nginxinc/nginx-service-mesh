package sidecar

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
)

// Pod defines the configuration of a single pod.
type Pod struct {
	ContainerPorts       map[string]string
	ParentName           string
	ParentType           string
	Name                 string
	Namespace            string
	ServiceAccountName   string
	PodIP                string
	IsIngressController  bool
	IsEgressController   bool
	Injected             bool
	DefaultEgressAllowed bool
}

// ToK8s returns the K8s resource associated with the API object.
func (p *Pod) ToK8s() *v1.Pod {
	controller := true

	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       p.ParentType,
					Controller: &controller,
				},
			},
			Labels: map[string]string{
				mesh.DeployLabel + p.ParentType: p.ParentName,
			},
		},
		Status: v1.PodStatus{
			PodIP: p.PodIP,
		},
	}
}
