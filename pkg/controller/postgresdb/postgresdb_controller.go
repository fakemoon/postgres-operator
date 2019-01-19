package postgresdb

import (
	"context"
	"strconv"
	"strings"

	fakemoonv1alpha1 "github.com/fakemoon/postgres-operator/pkg/apis/fakemoon/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const DEFAULT_POSTGRES_VERSION = "10"

var log = logf.Log.WithName("controller_postgresdb")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new PostgresDB Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePostgresDB{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("postgresdb-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource PostgresDB
	err = c.Watch(&source.Kind{Type: &fakemoonv1alpha1.PostgresDB{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner PostgresDB
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &fakemoonv1alpha1.PostgresDB{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePostgresDB{}

// ReconcilePostgresDB reconciles a PostgresDB object
type ReconcilePostgresDB struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a PostgresDB object and makes changes based on the state read
// and what is in the PostgresDB.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePostgresDB) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling PostgresDB")

	// Fetch the PostgresDB instance
	instance := &fakemoonv1alpha1.PostgresDB{}
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

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.makeDeploymentForPostgres(instance)
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Deployment")
		return reconcile.Result{}, err
	}
}

// makeDeploymentForPostgres returns a PostgresDB Deployment object
func (r *ReconcilePostgresDB) makeDeploymentForPostgres(p *fakemoonv1alpha1.PostgresDB) *appsv1.Deployment {
	ls := labelsForPostgres(p.Name)
	version := p.Spec.PostgresVersion
	imageName := "postgres"
	if checkPostgresVersion(version) {
		imageName += version
	} else {
		// using default
		log.Info("Invalid spec postgres_version. Using default = 10")
		imageName += DEFAULT_POSTGRES_VERSION
	}

	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: 1,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   imageName,
						Name:    "postgres",
						Command: []string{},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 5432,
							Name:          "postgres",
						}},
						Env:[]corev1.EnvVar{{
							Name:"POSTGRES_PASSWORD",
							Value:p.Spec.PostgresPassword,
						}},
					}},
				},
			},
		},
	}
	// Set Memcached instance as the owner and controller
	controllerutil.SetControllerReference(p, dep, r.scheme)
	return dep
}

// labelsForMemcached returns the labels for selecting the resources
// belonging to the given memcached CR name.
func labelsForPostgres(name string) map[string]string {
	return map[string]string{"app": "postgres", "postgres_cr": name}
}

// checkPostgresVersion returns true if version is a valid postgres version
// as example, not precisely
func checkPostgresVersion(version string) bool {
	versionList := strings.Split(version, ".")
	switch versionList[0] {
	case "11" :
		return true
	case "10" :
		return true
	case "9" :
		return true
	default:
		return false
	}
}
