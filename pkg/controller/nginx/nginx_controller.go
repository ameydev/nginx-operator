package nginx

import (
	"context"

	examplev1alpha1 "github.com/ameydev/nginx-operator/pkg/apis/example/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/extensions/v1beta1"
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

var log = logf.Log.WithName("controller_nginx")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Nginx Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNginx{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nginx-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Nginx
	err = c.Watch(&source.Kind{Type: &examplev1alpha1.Nginx{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to the resources that owned by the primary resource
	subresources := []runtime.Object{
		&appsv1.Deployment{},
		// &corev1.Service{},
	}

	for _, subresource := range subresources {
		err = c.Watch(&source.Kind{Type: subresource}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &examplev1alpha1.Nginx{},
		})
		if err != nil {
			return err
		}
	}
	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Nginx

	// err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &examplev1alpha1.Nginx{},
	// })
	// if err != nil {
	// 	return err
	// }

	return nil
}

// blank assignment to verify that ReconcileNginx implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNginx{}

// ReconcileNginx reconciles a Nginx object
type ReconcileNginx struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Nginx object and makes changes based on the state read
// and what is in the Nginx.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNginx) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Nginx")

	// Fetch the Nginx instance
	instance := &examplev1alpha1.Nginx{}
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

	return r.updateDeployment(instance, request)

}

func (r *ReconcileNginx) updateDeployment(instance *examplev1alpha1.Nginx, request reconcile.Request) (reconcile.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// Define a new Pod object
	dep := newDeploymentForCR(instance)

	// Set Nginx instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, dep, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Deployment already exists
	found := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, found)
	reqLogger.Info("found-replicas == ", found.Spec.Replicas)
	reqLogger.Info("dep-replicas == ", dep.Spec.Replicas)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Deployment")
		err := r.client.Create(context.TODO(), dep)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Deployment created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		// Compare existing deployment with required specs
		// Update deploying specs if required
		// Update the resource i.e deployment
		if found.Spec.Replicas == dep.Spec.Replicas {
			log.Info("No need to update replicas...")
			return reconcile.Result{}, err
		} else if found.Spec.Replicas != dep.Spec.Replicas {
			log.Info("Updating the found replicas by recreating")
			newDeploymentForCR(instance)
		}
		return reconcile.Result{}, err
	}

	// Deployment already exists - don't requeue
	reqLogger.Info("found-replicas == ", found.Spec.Replicas)
	reqLogger.Info("dep-replicas == ", dep.Spec.Replicas)

	log.Info("Skip reconcile: Deployment already exists")
	return reconcile.Result{}, nil

}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *examplev1alpha1.Nginx) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    cr.Spec.Name,
					Image:   cr.Spec.Image,
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}

func newDeploymentForCR(cr *examplev1alpha1.Nginx) *appsv1.Deployment {
	labels := map[string]string{
		"app": cr.Name,
	}
	// ls := labelsForMemcached(m.Name)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-deployment",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &cr.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: cr.Spec.Image,
						Name:  cr.Spec.Name,
						Ports: []corev1.ContainerPort{{
							ContainerPort: cr.Spec.Port,
						}},
					}},
				},
			},
		},
	}

	return dep
}

// func newServiceForCR(cr *examplev1alpha1.Nginx) *corev1.Sevice {
// 	labels := map[string]string{
// 		"app": cr.Name,
// 	}
// 	return &corev1.Service{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      cr.Name + "-service",
// 			Namespace: cr.Namespace,
// 			Labels:    labels,
// 		},
// 		Spec: corev1.PodSpec{
// 			Containers: []corev1.Container{
// 				{
// 					Name:    cr.Spec.Name,
// 					Image:   cr.Spec.Image,
// 					Command: []string{"sleep", "3600"},
// 				},
// 			},
// 		},
// 	}
// }
// func (r *ReconcileNginx) updatePod(instance *examplev1alpha1.Nginx) (reconcile.Result, error) {

// 	// Define a new Pod object
// 	pod := newPodForCR(instance)

// 	// Set Nginx instance as the owner and controller
// 	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
// 		return reconcile.Result{}, err
// 	}

// 	// Check if this Pod already exists
// 	found := &corev1.Pod{}
// 	err := r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
// 	if err != nil && errors.IsNotFound(err) {
// 		log.Info("Creating a new Pod")
// 		err := r.client.Create(context.TODO(), pod)
// 		if err != nil {
// 			return reconcile.Result{}, err
// 		}

// 		// Pod created successfully - don't requeue
// 		return reconcile.Result{}, nil
// 	} else if err != nil {
// 		return reconcile.Result{}, err
// 	}

// 	// Pod already exists - don't requeue
// 	log.Info("Skip reconcile: Pod already exists")
// 	return reconcile.Result{}, nil

// }
