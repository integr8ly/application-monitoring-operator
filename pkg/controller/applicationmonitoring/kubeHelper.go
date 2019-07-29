package applicationmonitoring

import (
	"fmt"
	"github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type KubeHelperImpl struct {
	k8client *kubernetes.Clientset
}

func NewKubeHelper() *KubeHelperImpl {
	config := config.GetConfigOrDie()

	k8client := kubernetes.NewForConfigOrDie(config)

	helper := new(KubeHelperImpl)
	helper.k8client = k8client
	return helper
}

// Watch secrets in the namespace that have a certain label
func (h *KubeHelperImpl) startSecretWatch(cr *v1alpha1.ApplicationMonitoring) (watch.Interface, error) {
	opts := v1.ListOptions{
		LabelSelector: fmt.Sprintf("monitoring-key=%s", cr.Spec.LabelSelector),
	}
	return h.k8client.CoreV1().Secrets(cr.Namespace).Watch(opts)
}