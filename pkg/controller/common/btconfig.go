package common

import (
	"sync"

	applicationmonitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
)

type BlackboxTargetsConfig struct {
	BTs map[string][]applicationmonitoringv1alpha1.BlackboxtargetData
}

var bts *BlackboxTargetsConfig
var once sync.Once

func GetBTConfig() *BlackboxTargetsConfig {
	once.Do(func() {
		bts = &BlackboxTargetsConfig{
			BTs: make(map[string][]applicationmonitoringv1alpha1.BlackboxtargetData),
		}
	})
	return bts
}

// Flatten returns the list flattened
func Flatten() []applicationmonitoringv1alpha1.BlackboxtargetData {
	bbtList := GetBTConfig()

	flattenedList := []applicationmonitoringv1alpha1.BlackboxtargetData{}
	for _, v := range bbtList.BTs {
		for i := range v {
			flattenedList = append(flattenedList, v[i])
		}
	}
	return flattenedList
}
