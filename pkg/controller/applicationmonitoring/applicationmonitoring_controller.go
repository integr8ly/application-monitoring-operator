package applicationmonitoring

import (
	"context"
	"fmt"
	"k8s.io/api/apps/v1beta1"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
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

const MonitoringFinalizerName = "monitoring.cleanup"

var log = logf.Log.WithName("controller_applicationmonitoring")

const (
	PhaseInstallPrometheusOperator int = iota
	PhaseWaitForOperator
	PhaseCreatePrometheusCRs
	PhaseCreateAlertManagerCrs
	PhaseCreateAux
	PhaseInstallGrafanaOperator
	PhaseCreateGrafanaCR
	PhaseFinalizer
	PhaseDone
)

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

	if instanceCopy.DeletionTimestamp != nil {
		return r.cleanup(instanceCopy)
	}

	switch instanceCopy.Status.Phase {
	case PhaseInstallPrometheusOperator:
		return r.installPrometheusOperator(instanceCopy)
	case PhaseWaitForOperator:
		return r.waitForPrometheusOperator(instanceCopy)
	case PhaseCreatePrometheusCRs:
		return r.createPrometheusCRs(instanceCopy)
	case PhaseCreateAlertManagerCrs:
		return r.createAlertManagerCRs(instanceCopy)
	case PhaseCreateAux:
		return r.createAux(instanceCopy)
	case PhaseInstallGrafanaOperator:
		return r.installGrafanaOperator(instanceCopy)
	case PhaseCreateGrafanaCR:
		return r.createGrafanaCR(instanceCopy)
	case PhaseFinalizer:
		return r.setFinalizer(instanceCopy)
	case PhaseDone:
		log.Info("Finished installing application monitoring")
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileApplicationMonitoring) installPrometheusOperator(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Install PrometheusOperator")

	err := r.createAdditionalScrapeConfig(cr)
	if err != nil {
		log.Error(err, "Failed to create additional scrape config")
		return reconcile.Result{}, err
	}

	for _, resourceName := range []string{PrometheusOperatorServiceAccountName, PrometheusOperatorName, PrometheusProxySecretsName, BlackboxExporterConfigmapName} {
		if _, err := r.createResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in InstallPrometheusOperator, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{}, err
		}
	}

	return r.updatePhase(cr, PhaseWaitForOperator)
}

func (r *ReconcileApplicationMonitoring) waitForPrometheusOperator(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Wait for Prometheus Operator")

	ready, err := r.getPrometheusOperatorReady(cr)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !ready {
		return reconcile.Result{RequeueAfter: time.Second * 10}, nil
	}

	log.Info("PrometheusOperator installation complete")
	return r.updatePhase(cr, PhaseCreatePrometheusCRs)
}

func (r *ReconcileApplicationMonitoring) createPrometheusCRs(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Create Prometheus CRs")

	// Create the route first and retrieve the host so that we can assign
	// it as the external url for the prometheus instance
	_, err := r.createResource(cr, PrometheusRouteName)
	if err != nil {
		return reconcile.Result{}, err
	}
	r.extraParams["prometheusHost"], err = r.getHostFromRoute(types.NamespacedName{Namespace: cr.Namespace, Name: PrometheusRouteName})
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, resourceName := range []string{PrometheusServiceAccountName, PrometheusServiceName, PrometheusCrName} {
		if _, err := r.createResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in CreatePrometheusCRs, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{}, err
		}
	}

	log.Info("Prometheus CRs installation complete")
	return r.updatePhase(cr, PhaseCreateAlertManagerCrs)
}

func (r *ReconcileApplicationMonitoring) createAlertManagerCRs(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Create AlertManager CRs")

	// Create the route first and retrieve the host so that we can assign
	// it as the external url for the alertmanager instance
	_, err := r.createResource(cr, AlertManagerRouteName)
	if err != nil {
		return reconcile.Result{}, err
	}
	r.extraParams["alertmanagerHost"], err = r.getHostFromRoute(types.NamespacedName{Namespace: cr.Namespace, Name: AlertManagerRouteName})
	if err != nil {
		return reconcile.Result{Requeue: true}, nil
	}

	for _, resourceName := range []string{AlertManagerServiceAccountName, AlertManagerServiceName, AlertManagerSecretName, AlertManagerProxySecretsName, AlertManagerCrName} {
		if _, err := r.createResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in CreateAlertManagerCRs, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{}, err
		}
	}

	log.Info("AlertManager CRs installation complete")
	return r.updatePhase(cr, PhaseCreateAux)
}

func (r *ReconcileApplicationMonitoring) createAux(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Create auxiliary resources")

	for _, resourceName := range []string{PrometheusServiceMonitorName, GrafanaServiceMonitorName, PrometheusRuleName} {
		if _, err := r.createResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in CreateAux, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{}, err
		}
	}

	log.Info("Auxiliary resources installation complete")
	return r.updatePhase(cr, PhaseInstallGrafanaOperator)
}

func (r *ReconcileApplicationMonitoring) installGrafanaOperator(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Install GrafanaOperator")

	for _, resourceName := range []string{GrafanaProxySecretName, GrafanaServiceName, GrafanaRouteName, GrafanaOperatorServiceAccountName, GrafanaOperatorRoleName, GrafanaOperatorRoleBindingName, GrafanaOperatorName} {
		if _, err := r.createResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in InstallGrafanaOperator, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{}, err
		}
	}

	log.Info("GrafanaOperator installation complete")

	// Give the operator some time to start
	return r.updatePhase(cr, PhaseCreateGrafanaCR)
}

func (r *ReconcileApplicationMonitoring) createGrafanaCR(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Create Grafana CR")

	for _, resourceName := range []string{GrafanaDataSourceName, GrafanaCrName} {
		if _, err := r.createResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in CreateGrafanaCR, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{}, err
		}
	}

	log.Info("Grafana CR installation complete")
	return r.updatePhase(cr, PhaseFinalizer)
}

// CreateResource Creates a generic kubernetes resource from a templates
func (r *ReconcileApplicationMonitoring) createResource(cr *applicationmonitoringv1alpha1.ApplicationMonitoring, resourceName string) (runtime.Object, error) {
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

// CreateResource Creates a generic kubernetes resource from a templates
func (r *ReconcileApplicationMonitoring) deleteResource(cr *applicationmonitoringv1alpha1.ApplicationMonitoring, resourceName string) error {
	templateHelper := newTemplateHelper(cr, r.extraParams)
	resourceHelper := newResourceHelper(cr, templateHelper)
	resource, err := resourceHelper.createResource(resourceName)
	if err != nil {
		return errors.Wrap(err, "createResource failed")
	}

	err = r.client.Delete(context.TODO(), resource)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return errors.Wrap(err, "error deleting resource")
		}
	}

	return nil
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

// Create a secret that contains additional prometheus scrape configurations
// Used to configure the blackbox exporter
func (r *ReconcileApplicationMonitoring) createAdditionalScrapeConfig(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) error {
	t := newTemplateHelper(cr, nil)
	job, err := t.loadTemplate(BlackboxExporterJobName)

	if err != nil {
		return errors.Wrap(err, "error loading blackbox exporter template")
	}

	scrapeConfigSecret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      ScrapeConfigSecretName,
			Namespace: cr.Namespace,
		},
		Data: map[string][]byte{
			"blackbox-exporter.yaml": []byte(job),
		},
	}

	err = controllerutil.SetControllerReference(cr, scrapeConfigSecret, r.scheme)
	if err != nil {
		return errors.Wrap(err, "error setting controller reference")
	}

	return r.client.Create(context.TODO(), scrapeConfigSecret)
}

func (r *ReconcileApplicationMonitoring) getPrometheusOperatorReady(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (bool, error) {
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

func (r *ReconcileApplicationMonitoring) setFinalizer(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Set Finalizer")

	if len(cr.Finalizers) == 0 {
		cr.Finalizers = append(cr.Finalizers, MonitoringFinalizerName)
	}

	return r.updatePhase(cr, PhaseDone)
}

func (r *ReconcileApplicationMonitoring) cleanup(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Cleanup")

	// Let the child operators finish their cleanup first
	for _, resourceName := range []string{GrafanaDataSourceName} {
		if err := r.deleteResource(cr, resourceName); err != nil {
			log.Info(fmt.Sprintf("Error in Cleanup, resourceName=%s : err=%s", resourceName, err))
			// Requeue so it can be attempted again
			return reconcile.Result{}, err
		}
	}

	// Wait to let the grafana operator remove the data source
	time.Sleep(time.Second * 5)

	// And finally remove the finalizer from the monitoring cr
	cr.Finalizers = nil
	err := r.client.Update(context.TODO(), cr)
	return reconcile.Result{}, err
}

func (r *ReconcileApplicationMonitoring) updatePhase(cr *applicationmonitoringv1alpha1.ApplicationMonitoring, phase int) (reconcile.Result, error) {
	cr.Status.Phase = phase
	err := r.client.Update(context.TODO(), cr)
	return reconcile.Result{}, err
}
