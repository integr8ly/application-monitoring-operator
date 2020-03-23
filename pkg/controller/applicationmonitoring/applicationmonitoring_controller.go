package applicationmonitoring

import (
	"context"
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	applicationmonitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	"github.com/integr8ly/application-monitoring-operator/pkg/controller/common"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	routev1 "github.com/openshift/api/route/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// MonitoringFinalizerName the name of the finalizer
const MonitoringFinalizerName = "monitoring.cleanup"

// ReconcilePauseSeconds the number of seconds to wait before running the reconcile loop
const ReconcilePauseSeconds = 30

var log = logf.Log.WithName("controller_applicationmonitoring")

// Constants used for the operator phases
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
	PhaseReconcileConfig
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
		helper:      NewKubeHelper(),
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

	return nil
}

var _ reconcile.Reconciler = &ReconcileApplicationMonitoring{}

// ReconcileApplicationMonitoring reconciles a ApplicationMonitoring object
type ReconcileApplicationMonitoring struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client      client.Client
	scheme      *runtime.Scheme
	helper      *KubeHelperImpl
	watch       watch.Interface
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
		if r.watch != nil {
			r.watch.Stop()
			r.watch = nil
		}

		return r.cleanup(instanceCopy)
	}

	if instanceCopy.Spec.PrometheusRetention == "" {
		instanceCopy.Spec.PrometheusRetention = "15d"
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
		return r.updatePhase(instanceCopy, PhaseReconcileConfig)
	case PhaseReconcileConfig:
		r.tryWatchAdditionalScrapeConfigs(instanceCopy)
		r.checkServiceAccountAnnotationsExist(instance)
		r.updateCRs(instance)
		return r.reconcileConfig(instanceCopy)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileApplicationMonitoring) reconcileConfig(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Reconciling Config")
	err := r.reconcileBlackboxExporterConfig(cr)
	if err != nil {
		return reconcile.Result{RequeueAfter: time.Second * ReconcilePauseSeconds}, err
	}

	err = r.syncBlackboxTargets(cr)
	return reconcile.Result{RequeueAfter: time.Second * ReconcilePauseSeconds}, err
}

func (r *ReconcileApplicationMonitoring) checkServiceAccountAnnotationsExist(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	key := "serviceaccounts.openshift.io/oauth-redirectreference.primary"
	namespace := cr.GetNamespace()

	result, err := r.ensureServiceAccountHasOauthAnnotation(cr, AlertManagerServiceAccountName, namespace, key, AlertManagerRouteName)
	if err != nil {
		return result, err
	}

	result, err = r.ensureServiceAccountHasOauthAnnotation(cr, PrometheusServiceAccountName, namespace, key, PrometheusRouteName)
	if err != nil {
		return result, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileApplicationMonitoring) updateCRs(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) {
	log.Info("Phase: Reconciling Config: updateCRs")

	var err error
	r.extraParams["prometheusHost"], err = r.getHostFromRoute(types.NamespacedName{Namespace: cr.Namespace, Name: PrometheusRouteName})
	r.extraParams["alertmanagerHost"], err = r.getHostFromRoute(types.NamespacedName{Namespace: cr.Namespace, Name: AlertManagerRouteName})
	if err != nil {
		log.Info(fmt.Sprintf("Error in updateCRs resourceName=%s, err=%s", PrometheusCrName, err))
	} else {
		prometheusInstance := &monitoringv1.Prometheus{}
		selector := client.ObjectKey{
			Namespace: cr.Namespace,
			Name:      ApplicationMonitoringName,
		}
		err := r.client.Get(context.TODO(), selector, prometheusInstance)
		if err != nil {
			log.Error(err, "error prometheus getting cr")
			return
		}
		r.updateCR(cr, PrometheusCrName, prometheusInstance.ResourceVersion)

		alertmanagerInstance := &monitoringv1.Alertmanager{}
		selector = client.ObjectKey{
			Namespace: cr.Namespace,
			Name:      ApplicationMonitoringName,
		}
		err = r.client.Get(context.TODO(), selector, alertmanagerInstance)
		if err != nil {
			log.Error(err, "error alertmanager getting cr")
			return
		}
		r.updateCR(cr, AlertManagerCrName, alertmanagerInstance.ResourceVersion)
	}
}

func (r *ReconcileApplicationMonitoring) updateCR(cr *applicationmonitoringv1alpha1.ApplicationMonitoring, crName string, resourceVersion string) {
	templateHelper := newTemplateHelper(cr, r.extraParams)
	resourceHelper := newResourceHelper(cr, templateHelper)
	resource, err := resourceHelper.createResource(crName)
	if err != nil {
		log.Error(err, "error updating cr")
		return
	}

	raw := resource.(*unstructured.Unstructured).UnstructuredContent()
	rawMetadata := raw["metadata"].(map[string]interface{})
	rawMetadata["resourceVersion"] = resourceVersion

	err = controllerutil.SetControllerReference(cr, resource.(metav1.Object), r.scheme)
	if err != nil {
		log.Error(err, fmt.Sprintf("error setting owner reference on %v", crName))
	}

	err = r.client.Update(context.TODO(), resource)
	if err != nil {
		log.Error(err, "error updating cr")
		return
	}
	log.Info("cr successfully updated")
}

func (r *ReconcileApplicationMonitoring) ensureServiceAccountHasOauthAnnotation(cr *applicationmonitoringv1alpha1.ApplicationMonitoring, sa string, ns string, key string, route string) (reconcile.Result, error) {
	instance := &corev1.ServiceAccount{}
	err := r.client.Get(context.TODO(), pkgclient.ObjectKey{Name: sa, Namespace: ns}, instance)
	if err != nil {
		log.Info(fmt.Sprintf("Error retrieving serviceaccount: %s : %s", sa, err))
		if strings.Contains(err.Error(), "not found") {
			log.Info(fmt.Sprintf("Creating serviceaccount: %s", sa))
			if _, err := r.createResource(cr, sa); err != nil {
				log.Info(fmt.Sprintf("Error creating serviceaccount, resourceName=%s : err=%s", sa, err))
				// Requeue so it can be attempted again
				return reconcile.Result{}, err
			}
		} else {
			return reconcile.Result{}, err
		}
	}

	if val, found := instance.Annotations[key]; found {
		log.Info(fmt.Sprintf("Service account: %s exists, annotations: %v", sa, instance.Annotations))
		log.Info(fmt.Sprintf("Key: %s exists. Val: %s Do nothing.", key, val))
	} else {
		log.Info(fmt.Sprintf("Service account: %s exists, annotations: %v", sa, instance.Annotations))
		log.Info(fmt.Sprintf("Key: %s does not exists. We need to add it.", key))
		if len(instance.Annotations) == 0 {
			instance.Annotations = map[string]string{}
		}
		instance.Annotations[key] = fmt.Sprintf("{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"%s\"}}", route)

		err = r.client.Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

// Update the blackbox exporter config if needed
// e.g. set tls cert config if using self signed certs
func (r *ReconcileApplicationMonitoring) reconcileBlackboxExporterConfig(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) error {
	// Read the blackbox exporter configmap, which should already exist
	blackboxExporterConfigmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "blackbox-exporter-config",
			Namespace: cr.Namespace,
		},
	}

	ctx := context.TODO()

	if err := r.client.Get(ctx, client.ObjectKey{Name: blackboxExporterConfigmap.Name, Namespace: blackboxExporterConfigmap.Namespace}, blackboxExporterConfigmap); err != nil {
		log.Error(err, "client.Get")
		return fmt.Errorf("error getting blackbox exporter configmap.: %s", err.Error())
	}

	// Build the full blackbox config based on the AMO CR config
	if r.extraParams == nil {
		r.extraParams = map[string]string{}
	}
	r.extraParams["selfSignedCerts"] = strconv.FormatBool(cr.Spec.SelfSignedCerts)
	templateHelper := newTemplateHelper(cr, r.extraParams)
	blackboxExporterConfig, err := templateHelper.loadTemplate("blackbox/blackbox-exporter-config")
	if err != nil {
		log.Error(err, "templateHelper.loadTemplate")
		return fmt.Errorf("error loading template: %s", err.Error())
	}

	// Update the configmap if needed
	if blackboxExporterConfigmap.Data["blackbox.yml"] != string(blackboxExporterConfig) {
		blackboxExporterConfigmap.Data = map[string]string{
			"blackbox.yml": string(blackboxExporterConfig),
		}
		if err := r.client.Update(ctx, blackboxExporterConfigmap); err != nil {
			log.Error(err, "serverClient.Update")
			return fmt.Errorf("error updating blackbox exporter configmap: %s", err.Error())
		}
		pods := &corev1.PodList{}
		opts := []client.ListOption{
			client.InNamespace(cr.Namespace),
			client.MatchingLabels{"app": "prometheus", "prometheus": "application-monitoring"},
		}
		err := r.client.List(ctx, pods, opts...)
		if err != nil {
			return fmt.Errorf("failed to list pods: %s", err.Error())
		}
		if len(pods.Items) > 0 {
			log.Info("Attempting to delete pod to reload config")
			err = r.client.Delete(ctx, &pods.Items[0])
			if err != nil {
				return fmt.Errorf("error deleting pod: %s", err.Error())
			}
		}
	}
	return nil
}

func (r *ReconcileApplicationMonitoring) syncBlackboxTargets(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) error {
	log.Info("Phase: Reconciling Config syncBlackboxTargets")

	previous := cr.Status.LastBlackboxConfig

	flatList := common.Flatten()
	current := joinQuote(flatList)

	check, hash := r.hasBlackboxTargetsListChanged(previous, current)

	if check {
		log.Info(fmt.Sprintf("Phase: Reconciling Config syncBlackboxTargets hasBlackboxTargetsListChanged: %v hash: %v", check, hash))
		err := r.createOrUpdateAdditionalScrapeConfig(cr)

		return err
	}
	return nil
}

func (r *ReconcileApplicationMonitoring) tryWatchAdditionalScrapeConfigs(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) {
	// Watch already set
	if r.watch != nil {
		return
	}

	watch, err := r.watchAdditionalScrapeConfigs(cr)
	if err != nil {
		log.Error(err, "error setting up watch for additional scrape config")
		return
	}

	r.watch = watch
	log.Info("watching additional scrape configs")
}

// Watches the additional scrape config secret for modifications and reconciles them into the prometheus
// configuration
func (r *ReconcileApplicationMonitoring) watchAdditionalScrapeConfigs(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (watch.Interface, error) {
	if cr.Spec.AdditionalScrapeConfigSecretName == "" {
		return nil, nil
	}

	events, err := r.helper.startSecretWatch(cr)
	if err != nil {
		return nil, err
	}

	go func() {
		for update := range events.ResultChan() {
			secret := update.Object.(*corev1.Secret)
			if update.Type != watch.Error && secret.Name == cr.Spec.AdditionalScrapeConfigSecretName {
				log.Info(fmt.Sprintf("watch event of type '%v' received for additional scrape config", update.Type))
				err = r.createOrUpdateAdditionalScrapeConfig(cr)
				if err != nil {
					log.Error(err, "error updating additional scrape config")
				}
			}
		}
		log.Info("watch ended for additional scrape config")
		r.watch = nil
	}()

	return events, nil
}

func (r *ReconcileApplicationMonitoring) installPrometheusOperator(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (reconcile.Result, error) {
	log.Info("Phase: Install PrometheusOperator")

	err := r.createOrUpdateAdditionalScrapeConfig(cr)
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

	for _, resourceName := range []string{GrafanaProxySecretName, GrafanaOperatorServiceAccountName, GrafanaOperatorRoleName, GrafanaOperatorRoleBindingName, GrafanaOperatorName} {
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
	err = controllerutil.SetControllerReference(cr, resource.(metav1.Object), r.scheme)
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
func (r *ReconcileApplicationMonitoring) readAdditionalScrapeConfigSecret(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) ([]byte, bool) {
	if cr.Spec.AdditionalScrapeConfigSecretName == "" || cr.Spec.AdditionalScrapeConfigSecretKey == "" {
		log.Info("no additional scrape config specified")
		return nil, false
	}

	selector := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      cr.Spec.AdditionalScrapeConfigSecretName,
	}

	additionalScrapeConfig := corev1.Secret{}
	err := r.client.Get(context.TODO(), selector, &additionalScrapeConfig)
	if err != nil {
		log.Info(fmt.Sprintf("can't find secret '%v'", cr.Spec.AdditionalScrapeConfigSecretName))
		return nil, false
	}

	if val, ok := additionalScrapeConfig.Data[cr.Spec.AdditionalScrapeConfigSecretKey]; ok {
		return val, true
	}

	return nil, false
}

// Create a secret that contains additional prometheus scrape configurations
// Used to configure the blackbox exporter
func (r *ReconcileApplicationMonitoring) createOrUpdateAdditionalScrapeConfig(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) error {
	t := newTemplateHelper(cr, nil)
	job, err := t.loadTemplate(BlackboxExporterJobName)

	if err != nil {
		return errors.Wrap(err, "error loading blackbox exporter template")
	}

	additionalConfig, found := r.readAdditionalScrapeConfigSecret(cr)
	if found {
		job = append(job, []byte("\n")...)
		job = append(job, additionalConfig...)
	}

	selector := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      ScrapeConfigSecretName,
	}

	update := true
	scrapeConfigSecret := corev1.Secret{}

	// Check if the secret already exists
	err = r.client.Get(context.TODO(), selector, &scrapeConfigSecret)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
		update = false
	}

	scrapeConfigSecret.ObjectMeta = metav1.ObjectMeta{
		Name:      ScrapeConfigSecretName,
		Namespace: cr.Namespace,
	}

	scrapeConfigSecret.Data = map[string][]byte{
		"integreatly.yaml": []byte(job),
	}

	flatList := common.Flatten()
	cr.Status.LastBlackboxConfig = fmt.Sprintf("%x", md5.Sum([]byte(joinQuote(flatList))))

	err = r.client.Update(context.TODO(), cr)
	if err != nil {
		return err
	}

	if update {
		return r.client.Update(context.TODO(), &scrapeConfigSecret)
	} else {
		err = controllerutil.SetControllerReference(cr, &scrapeConfigSecret, r.scheme)
		if err != nil {
			return errors.Wrap(err, "error setting controller reference")
		}

		return r.client.Create(context.TODO(), &scrapeConfigSecret)
	}
}

func (r *ReconcileApplicationMonitoring) getPrometheusOperatorReady(cr *applicationmonitoringv1alpha1.ApplicationMonitoring) (bool, error) {
	resource := appsv1.Deployment{}

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
		r.client.Update(context.TODO(), cr)
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

func (r *ReconcileApplicationMonitoring) hasBlackboxTargetsListChanged(previous string, current string) (bool, string) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(current)))
	return hash != previous, hash
}
