package blackboxtarget

import (
	"context"
	"fmt"

	applicationmonitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	"github.com/integr8ly/application-monitoring-operator/pkg/controller/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_blackboxtarget")

// MonitoringFinalizerName the name of the finaliser
const MonitoringFinalizerName = "monitoring.cleanup"

// The phases for the BlackboxTarget CR types
const (
	PhaseFinalizer int = iota
	PhaseDone
	PhaseReconcileConfig
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new BlackboxTarget Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBlackboxTarget{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("blackboxtarget-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource BlackboxTarget
	err = c.Watch(&source.Kind{Type: &applicationmonitoringv1alpha1.BlackboxTarget{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileBlackboxTarget implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileBlackboxTarget{}

// ReconcileBlackboxTarget reconciles a BlackboxTarget object
type ReconcileBlackboxTarget struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a BlackboxTarget object and makes changes based on the state read
// and what is in the BlackboxTarget.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileBlackboxTarget) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling BlackboxTarget")

	// Fetch the BlackboxTarget instance
	instance := &applicationmonitoringv1alpha1.BlackboxTarget{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	switch instance.Status.Phase {
	case PhaseFinalizer:
		return r.setFinalizer(instance)
	case PhaseDone:
		reqLogger.Info(fmt.Sprintf("BlackboxTarget phaseDone CR:%s Phase: %v", instance.ObjectMeta.Name, instance.Status.Phase))
		return r.updatePhase(instance, PhaseReconcileConfig)
	case PhaseReconcileConfig:
		return r.reconcileConfig(instance)
	}

	return reconcile.Result{}, nil
}

// reconcileConfig not sure if these CRs need to be watched for changes, presume I'll do it here
func (r *ReconcileBlackboxTarget) reconcileConfig(cr *applicationmonitoringv1alpha1.BlackboxTarget) (reconcile.Result, error) {
	log.Info(fmt.Sprintf("BlackboxTarget reconcileConfig CR:%s Phase: %v", cr.ObjectMeta.Name, cr.Status.Phase))

	bbtList := common.GetBTConfig()
	crName := cr.ObjectMeta.Name
	// Remove the finalizer so the CR can be deleted
	if cr.DeletionTimestamp != nil {
		// Remove the blackboxtargets assocated with this CR as its being deleted
		delete(bbtList.BTs, crName)
		return reconcile.Result{}, r.removeFinalizer(cr)
	}

	// Overwrite the blackboxtargets stored for a particular CR
	bbtList.BTs[crName] = cr.Spec.BlackboxTargets

	return reconcile.Result{}, nil
}

// setFinalizer Sets the finaliser on the BlackboxTarget CR
func (r *ReconcileBlackboxTarget) setFinalizer(cr *applicationmonitoringv1alpha1.BlackboxTarget) (reconcile.Result, error) {
	log.Info(fmt.Sprintf("BlackboxTarget setFinalizer CR:%s", cr.ObjectMeta.Name))

	if len(cr.Finalizers) == 0 {
		cr.Finalizers = append(cr.Finalizers, MonitoringFinalizerName)
	}

	return r.updatePhase(cr, PhaseDone)
}

// updatePhase updates the CR.Status.Phase
func (r *ReconcileBlackboxTarget) updatePhase(cr *applicationmonitoringv1alpha1.BlackboxTarget, phase int) (reconcile.Result, error) {
	log.Info(fmt.Sprintf("BlackboxTarget updatePhase CR:%s Phase: %v", cr.ObjectMeta.Name, phase))

	cr.Status.Phase = phase
	err := r.client.Update(context.TODO(), cr)
	return reconcile.Result{}, err
}

func (r *ReconcileBlackboxTarget) removeFinalizer(cr *applicationmonitoringv1alpha1.BlackboxTarget) error {
	log.Info(fmt.Sprintf("BlackboxTarget removeFinalizer CR:%s", cr.ObjectMeta.Name))

	cr.Finalizers = nil
	return r.client.Update(context.TODO(), cr)
}
