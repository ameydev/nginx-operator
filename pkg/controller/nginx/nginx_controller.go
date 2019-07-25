package nginx

import (
	"context"
	"fmt"
	"reflect"

	examplev1alpha1 "github.com/ameydev/nginx-operator/pkg/apis/example/v1alpha1"
	// "k8s.io/api/apps/v1beta1"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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
		&corev1.Service{},
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

	res, err := r.updateSecret(instance, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	res, err = r.updateDeployment(instance, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	res, err = r.updateService(instance, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	return res, err

}

func (r *ReconcileNginx) updateSecret(instance *examplev1alpha1.Nginx, request reconcile.Request) (reconcile.Result, error) {

	// reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// Define a new Pod object
	secret := newSecretForCR(instance)

	// Set Nginx instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, secret, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Deployment already exists
	found := &corev1.Secret{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
	fmt.Println("Found secret ===========")
	// fmt.Println(found)
	username := instance.Spec.Username
	// e_usernmae := b64.StdEncoding.EncodeToString([]byte(username))
	eUsernmae := []byte(username)
	fmt.Println("Instance Username", username)
	fmt.Println("Eusername ==== ", eUsernmae)
	fmt.Println("Eusername in byte ==== ", []byte(eUsernmae))
	fmt.Println("Found username ==== ", found.Data["username"])

	// if found.Data["username"] != []byte(e_usernmae) {
	if found.Data["username"] != nil {
		if reflect.DeepEqual(found.Data["username"], []byte(eUsernmae)) == false {
			found.Data["username"] = eUsernmae

			fmt.Println("Username updated as ==> ", username)
			err = r.client.Update(context.TODO(), found)
			if err != nil {
				fmt.Println("Failed to update Secret: %v\n", err)
				return reconcile.Result{}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true}, nil
		}
	}
	password := instance.Spec.Password
	ePassword := []byte(password)

	fmt.Println("Instance Password", password)
	fmt.Println("E_pass ==== ", ePassword)
	fmt.Println("E_pass in byte ==== ", []byte(ePassword))
	fmt.Println("Found passwrd ==== ", found.Data["password"])

	if found.Data["password"] != nil {
		if reflect.DeepEqual(found.Data["password"], ePassword) == false {
			found.Data["password"] = ePassword

			fmt.Println("Password updated as ==> ", password)
			err = r.client.Update(context.TODO(), found)
			if err != nil {
				fmt.Println("Failed to update Secret: %v\n", err)
				return reconcile.Result{}, err
			}
			// Spec updated - return and requeue
			return reconcile.Result{Requeue: true}, nil
		}
	}

	if err != nil && errors.IsNotFound(err) {
		fmt.Println("Creating a new secret")
		err := r.client.Create(context.TODO(), secret)
		if err != nil {
			return reconcile.Result{}, err
		}

		// secret created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {

		return reconcile.Result{}, err
	}

	log.Info("Skip reconcile: Secret already exists")
	return reconcile.Result{}, nil

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
	// reqLogger.Info("found-replicas == ", found.Spec.Replicas)Replicas
	// reqLogger.Info("dep-replicas == ", dep.Spec.Replicas)
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
		// if *found.Spec.Replicas == dep.Spec.Replicas {
		// 	log.Info("No need to update replicas...")
		// 	return reconcile.Result{}, err
		// } else if found.Spec.Replicas != instance.Spec.Replicas {
		// 	log.Info("=======Updating the found replicas by recreating=======")
		// 	newDeploymentForCR(instance)
		// }
		return reconcile.Result{}, err
	}

	// Ensure the deployment Count is the same as the spec
	count := instance.Spec.Replicas
	if *found.Spec.Replicas != count {
		found.Spec.Replicas = &count
		// reqLogger.Info("found-count in new method == ", &count)
		fmt.Println("found-count in new method == ", count)
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			reqLogger.Info("Failed to update Deployment: %v\n", err)
			return reconcile.Result{}, err
		}
		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// Deployment already exists - don't requeue
	// reqLogger.Info("found-replicas == ", found.Spec.Replicas)
	// reqLogger.Info("dep-replicas == ", dep.Spec.Replicas)

	log.Info("Skip reconcile: Deployment already exists")
	return reconcile.Result{}, nil

}

func (r *ReconcileNginx) updateService(instance *examplev1alpha1.Nginx, request reconcile.Request) (reconcile.Result, error) {

	// reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// Define a new Pod object
	service := newServiceForCR(instance)

	// Set Nginx instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Deployment already exists
	found := &corev1.Service{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		fmt.Println("Creating a new Service")
		err := r.client.Create(context.TODO(), service)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Service created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		// Compare existing deployment with required specs
		// Update deploying specs if required
		// Update the resource i.e deployment
		// if *found.Spec.Replicas == dep.Spec.Replicas {
		// 	log.Info("No need to update replicas...")
		// 	return reconcile.Result{}, err
		// } else if found.Spec.Replicas != instance.Spec.Replicas {
		// 	log.Info("=======Updating the found replicas by recreating=======")
		// 	newDeploymentForCR(instance)
		// }
		return reconcile.Result{}, err
	}

	// Ensure the Port is the same as the spec
	// port := instance.Spec.Port
	// if *found.Spec.ServicePort.Ports[0] != port {
	// 	found.Spec.ServicePort.Ports[0] = &port
	// 	// reqLogger.Info("found-count in new method == ", &count)
	// 	fmt.Println("found-port in new method == ", port)
	// 	err = r.client.Update(context.TODO(), found)
	// 	if err != nil {
	// 		reqLogger.Info("Failed to update Deployment: %v\n", err)
	// 		return reconcile.Result{}, err
	// 	}
	// 	// Spec updated - return and requeue
	// 	return reconcile.Result{Requeue: true}, nil
	// }

	// Deployment already exists - don't requeue
	// reqLogger.Info("found-replicas == ", found.Spec.Replicas)
	// reqLogger.Info("dep-replicas == ", dep.Spec.Replicas)

	log.Info("Skip reconcile: Service already exists")
	return reconcile.Result{}, nil

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
			Strategy: appsv1.DeploymentStrategy{
				Type: "Recreate",
			},
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
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: cr.Spec.Port,
							},
						},
						VolumeMounts: []corev1.VolumeMount{{

							Name:      "secret-volume",
							MountPath: "/etc/secret-volume",
						},
						},
					}},
					// spec.template.spec.volumes[0].secret.secretName:
					Volumes: []corev1.Volume{
						{
							Name: "secret-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: cr.Spec.SecretName,
								},
							},
						},
					},
				},
			},
		},
	}

	return dep
}

func newServiceForCR(cr *examplev1alpha1.Nginx) *corev1.Service {
	labels := map[string]string{
		"app": cr.Name,
	}
	crport := cr.Spec.Port

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-service",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     "LoadBalancer",
			Selector: map[string]string{"app": cr.Name},
			Ports: []corev1.ServicePort{{
				Port:       cr.Spec.Port,
				TargetPort: intstr.FromInt(int(crport)),
				Protocol:   "TCP",
				Name:       "http",
			}},
		},
	}

	return service

}

func newSecretForCR(cr *examplev1alpha1.Nginx) *corev1.Secret {
	labels := map[string]string{
		"app": cr.Name,
	}
	// crport := cr.Spec.Port

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Spec.SecretName,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		StringData: map[string]string{
			"username": cr.Spec.Username,
			"password": cr.Spec.Password,
		},
	}
	return secret
}
