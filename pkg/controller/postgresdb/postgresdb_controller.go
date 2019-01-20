package postgresdb

import (
	"context"
	"k8s.io/apimachinery/pkg/api/resource"
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
const DEFAULT_POSTGRES_STORAGE = "5Gi"

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
	found := &appsv1.StatefulSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		// dep := r.makeDeploymentForPostgres(instance)
		// Define a new stateful set
		stf := r.makeStatefulSetForPostgres(instance)
		reqLogger.Info("Creating a new StatefulSet", "StatefulSet.Namespace", stf.Namespace, "StatefulSet.Name", stf.Name)
		err = r.client.Create(context.TODO(), stf)
		if err != nil {
			reqLogger.Error(err, "Failed to create new StatefulSet", "StatefulSet.Namespace", stf.Namespace, "StatefulSet.Name", stf.Name)
			return reconcile.Result{}, err
		}
		// StatefulSet created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get StatefulSet")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}


// makeStatefulSetForPostgres returns a PostgresDB StatefulSet object
func (r *ReconcilePostgresDB) makeStatefulSetForPostgres(p *fakemoonv1alpha1.PostgresDB) *appsv1.StatefulSet {
	ls := labelsForPostgres(p.Name)
	version := p.Spec.PostgresVersion
	imageName := "postgres:"
	if checkPostgresVersion(version) {
		imageName += version
	} else {
		// using default
		log.Info("Invalid spec postgres_version. Using default = 10")
		imageName += DEFAULT_POSTGRES_VERSION
	}
	quantity, err := resource.ParseQuantity(p.Spec.Storage)
	if err != nil {
		//use default storage size
		log.Info("Invalid spec storage. Using default = 5Gi")
		quantity, _ = resource.ParseQuantity(DEFAULT_POSTGRES_STORAGE)
	}
	// for non-cluster postgres deploy, only 1 replica
	var replicas int32 = 1

	stf := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
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
						VolumeMounts:[]corev1.VolumeMount{{
							Name:"postgres_data_vol",
							MountPath:"/var/lib/postgresql/data",
						}},
					}},
				},
			},
			VolumeClaimTemplates:[]corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{
					Name:"postgres_data_vol",
				},
				Spec:corev1.PersistentVolumeClaimSpec{
					AccessModes:[]corev1.PersistentVolumeAccessMode{
						"ReadWriteOnce",
					},
					Resources:corev1.ResourceRequirements{
						Requests:corev1.ResourceList{
							"storage": quantity,
						},
					},
					StorageClassName:&p.Name,
				},
			}},
		},
	}
	// Set Postgres instance as the owner and controller
	controllerutil.SetControllerReference(p, stf, r.scheme)
	return stf
}

// labelsForPostgres returns the labels for selecting the resources
// belonging to the given postgres CR name.
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
