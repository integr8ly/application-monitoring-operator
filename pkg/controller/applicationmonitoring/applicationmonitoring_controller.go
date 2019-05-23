package applicationmonitoring

import (
	"context"
	"fmt"
	"time"

	"k8s.io/api/apps/v1beta1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	applicationmonitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_applicationmonitoring")

const (
	PhaseInstallPrometheusOperator int = iota
	PhaseWaitForOperator
	PhaseCreatePrometheusCRs
	PhaseCreateAlertManagerCrs
	PhaseCreateAux
	PhaseInstallGrafanaOperator
	PhaseCreateGrafanaCR
	PhaseDone
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ApplicationMonitoring Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileApplicationMonitoring{
		client:      mgr.GetClient(),
		scheme:      mgr.GetScheme(),
		extraParams: make(map[string]string),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("applicationmonitoring-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ApplicationMonitoring
	err = c.Watch(&source.Kind{Type: &applicationmonitoringv1alpha1.ApplicationMonitoring{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner ApplicationMonitoring
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &applicationmonitoringv1alpha1.ApplicationMonitoring{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileApplicationMonitoring{}

// ReconcileApplicationMonitoring reconciles a ApplicationMonitoring object
type ReconcileApplicationMonitoring struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client      client.Client
	scheme      *runtime.Scheme
	extraParams map[string]string
}

// Reconcile reads that state of the cluster for a ApplicationMonitoring object and makes changes based on the state read
// and what is in the ApplicationMonitoring.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileApplicationMonitoring) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ApplicationMonitoring")

	// Fetch the ApplicationMonitoring instance
	instance := &applicationmonitoringv1alpha1.ApplicationMonitoring{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	instanceCopy := instance.DeepCopy()

	switch instanceCopy.Status.Phase {
	case PhaseInstallPrometheusOperator:
		return r.InstallPrometheusOperator(instanceCopy)
	case PhaseWaitForOperator:
		return r.WaitForPrometheusOperator(instanceCopy)
	case PhaseCreatePrometheusCRs:
		return r.CreatePrometheusCRs(instanceCopy)
	case PhaseCreateAlertManagerCrs:
		return r.CreateAlertManagerCRs(instanceCopy)
	case PhaseCreateAux:
		return r.CreateAux(instanceCopy)
	case PhaseInstallGrafanaOperator:
		return r.InstallGrafanaOperator(instanceCopy)
	case PhaseCreateGrafanaCR:
		return r.CreateGrafanaCR(instanceCopy)
	case PhaseDone:
		log.Info("Finished installing application monitoring")
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileApplicationMonitoring) InstallPrometheusOperator(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Install PrometheusOperator")

	for _, resourceName := range []string{PrometheusOperatorServiceAccountName, PrometheusOperatorName, PrometheusProxySecretsName} {
		if _, err := r.CreateResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in InstallPrometheusOperator, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{RequeueAfter: time.Second * 10}, r.UpdatePhase(cr, PhaseWaitForOperator)
}

func (r *ReconcileApplicationMonitoring) WaitForPrometheusOperator(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Wait for Prometheus Operator")

	ready, err := r.GetPrometheusOperatorReady(cr)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !ready {
		return reconcile.Result{RequeueAfter: time.Second * 10}, nil
	}

	log.Info("PrometheusOperator installation complete")
	return reconcile.Result{Requeue: true}, r.UpdatePhase(cr, PhaseCreatePrometheusCRs)
}

func (r *ReconcileApplicationMonitoring) CreatePrometheusCRs(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Create Prometheus CRs")

	// Create the route first and retrieve the host so that we can assign
	// it as the external url for the prometheus instance
	_, err := r.CreateResource(cr, PrometheusRouteName)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	r.extraParams["prometheusHost"], err = r.getHostFromRoute(types.NamespacedName{Namespace: cr.Namespace, Name: PrometheusRouteName})
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	for _, resourceName := range []string{PrometheusServiceAccountName, PrometheusServiceName, PrometheusCrName} {
		if _, err := r.CreateResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in CreatePrometheusCRs, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{Requeue: true}, err
		}
	}

	log.Info("Prometheus CRs installation complete")
	return reconcile.Result{RequeueAfter: time.Second * 10}, r.UpdatePhase(cr, PhaseCreateAlertManagerCrs)
}

func (r *ReconcileApplicationMonitoring) CreateAlertManagerCRs(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Create AlertManager CRs")

	// Create the route first and retrieve the host so that we can assign
	// it as the external url for the alertmanager instance
	_, err := r.CreateResource(cr, AlertManagerRouteName)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	r.extraParams["alertmanagerHost"], err = r.getHostFromRoute(types.NamespacedName{Namespace: cr.Namespace, Name: AlertManagerRouteName})
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	for _, resourceName := range []string{AlertManagerServiceAccountName, AlertManagerServiceName, AlertManagerSecretName, AlertManagerProxySecretsName, AlertManagerCrName} {
		if _, err := r.CreateResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in CreateAlertManagerCRs, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{Requeue: true}, err
		}
	}

	log.Info("AlertManager CRs installation complete")
	return reconcile.Result{RequeueAfter: time.Second * 10}, r.UpdatePhase(cr, PhaseCreateAux)
}

func (r *ReconcileApplicationMonitoring) CreateAux(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Create auxiliary resources")

	for _, resourceName := range []string{PrometheusServiceMonitorName, GrafanaServiceMonitorName, PrometheusRuleName} {
		if _, err := r.CreateResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in CreateAux, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{Requeue: true}, err
		}
	}

	log.Info("Auxiliary resources installation complete")
	return reconcile.Result{RequeueAfter: time.Second * 10}, r.UpdatePhase(cr, PhaseInstallGrafanaOperator)
}

func (r *ReconcileApplicationMonitoring) InstallGrafanaOperator(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Install GrafanaOperator")

	for _, resourceName := range []string{GrafanaProxySecretName, GrafanaOperatorServiceAccountName, GrafanaOperatorRoleName, GrafanaOperatorRoleBindingName, GrafanaOperatorName} {
		if _, err := r.CreateResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in InstallGrafanaOperator, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{Requeue: true}, err
		}
	}

	log.Info("GrafanaOperator installation complete")

	// Give the operator some time to start
	return reconcile.Result{RequeueAfter: time.Second * 10}, r.UpdatePhase(cr, PhaseCreateGrafanaCR)
}

func (r *ReconcileApplicationMonitoring) CreateGrafanaCR(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Create Grafana CR")

	for _, resourceName := range []string{GrafanaCrName} {
		if _, err := r.CreateResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in CreateGrafanaCR, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{Requeue: true}, err
		}
	}

	log.Info("Grafana CR installation complete")
	return reconcile.Result{RequeueAfter: time.Second * 10}, r.UpdatePhase(cr, PhaseDone)
}

// CreateResource Creates a generic kubernetes resource from a templates
func (r *ReconcileApplicationMonitoring) CreateResource(cr *applicationmonitoringv1alpha1.ApplicationMonitoring, resourceName string) (runtime.Object, error) {
	templateHelper := newTemplateHelper(cr, r.extraParams)
	resourceHelper := newResourceHelper(cr, templateHelper)
	resource, err := resourceHelper.createResource(resourceName)

	if err != nil {
		return nil, errors.Wrap(err, "createResource failed")
	}

	// Set the CR as the owner of this resource so that when
	// the CR is deleted this resource also gets removed
	err = controllerutil.SetControllerReference(cr, resource.(v1.Object), r.scheme)
	if err != nil {
		return nil, errors.Wrap(err, "error setting controller reference")
	}

	err = r.client.Create(context.TODO(), resource)
	if err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return nil, errors.Wrap(err, "error creating resource")
		}
	}

	return resource, nil
}

func (r *ReconcileApplicationMonitoring) getHostFromRoute(namespacedName types.NamespacedName) (string, error) {

	route := &routev1.Route{}

	err := r.client.Get(context.TODO(), namespacedName, route)
	if err != nil {
		return "", err
	}

	host := route.Spec.Host

	if host == "" {
		return "", errors.New("Error getting host from route: host value empty")
	}
	return host, nil
}

func (r *ReconcileApplicationMonitoring) GetPrometheusOperatorReady(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (bool, error) {
	resource := v1beta1.Deployment{}

	selector := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      PrometheusOperatorName,
	}

	err := r.client.Get(context.TODO(), selector, &resource)
	if err != nil {
		return false, err
	}

	return resource.Status.ReadyReplicas == 1, nil
}

func (r *ReconcileApplicationMonitoring) UpdatePhase(cr *applicationmonitoringv1alpha1.ApplicationMonitoring, phase int) error {
	cr.Status.Phase = phase
	return r.client.Update(context.TODO(), cr)
}
