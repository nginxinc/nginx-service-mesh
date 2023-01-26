package config_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nginxinc/nginx-service-mesh/pkg/config"
)

var _ = Describe("Pod", func() {
	pod := config.Pod{
		ParentName:     "test",
		ParentType:     "deployment",
		Name:           "test-84b46c648f-tfzrs",
		Namespace:      "default",
		PodIP:          "1.2.3.4",
		Injected:       true,
		ContainerPorts: map[string]string{"80": "8888"},
	}

	It("can convert to a k8s object", func() {
		controller := true
		expPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-84b46c648f-tfzrs",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind:       "deployment",
						Controller: &controller,
					},
				},
				Labels: map[string]string{
					"nsm.nginx.com/deployment": "test",
				},
			},
			Status: v1.PodStatus{
				PodIP: "1.2.3.4",
			},
		}
		Expect(reflect.DeepEqual(pod.ToK8s(), expPod)).To(BeTrue())
	})
})
