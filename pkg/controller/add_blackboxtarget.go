package controller

import (
	"github.com/integr8ly/application-monitoring-operator/pkg/controller/blackboxtarget"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, blackboxtarget.Add)
}
