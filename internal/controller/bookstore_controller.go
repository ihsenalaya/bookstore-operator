package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	storev1alpha1 "bookstore-operator/api/v1alpha1"
)

// BookStoreReconciler reconciles a BookStore object
type BookStoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=store.mylab.local,resources=bookstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=store.mylab.local,resources=bookstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=store.mylab.local,resources=bookstores/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *BookStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Lire le BookStore
	var bookstore storev1alpha1.BookStore
	if err := r.Get(ctx, req.NamespacedName, &bookstore); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("BookStore resource not found")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch BookStore")
		return ctrl.Result{}, err
	}

	// 2. Valeurs par défaut
	replicas := int32(1)
	if bookstore.Spec.Replicas != nil {
		replicas = *bookstore.Spec.Replicas
	}

	image := bookstore.Spec.Image
	if image == "" {
		image = "nginx:1.25"
	}

	port := bookstore.Spec.Port
	if port == 0 {
		port = 80
	}

	labels := map[string]string{
		"app":       "bookstore",
		"bookstore": bookstore.Name,
	}

	// 3. Deployment
	deploymentName := fmt.Sprintf("%s-deployment", bookstore.Spec.Name)
	var deployment appsv1.Deployment

	err := r.Get(ctx, types.NamespacedName{
		Name:      deploymentName,
		Namespace: bookstore.Namespace,
	}, &deployment)

	if err != nil && apierrors.IsNotFound(err) {
		deployment = appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName,
				Namespace: bookstore.Namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "bookstore",
								Image: image,
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: port,
									},
								},
							},
						},
					},
				},
			},
		}

		if err := controllerutil.SetControllerReference(&bookstore, &deployment, r.Scheme); err != nil {
			logger.Error(err, "unable to set owner reference on Deployment")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, &deployment); err != nil {
			logger.Error(err, "unable to create Deployment")
			return ctrl.Result{}, err
		}

		logger.Info("Deployment created", "name", deployment.Name)

	} else if err != nil {
		logger.Error(err, "unable to get Deployment")
		return ctrl.Result{}, err

	} else {
		updated := false

		if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != replicas {
			deployment.Spec.Replicas = &replicas
			updated = true
		}

		if len(deployment.Spec.Template.Spec.Containers) > 0 {
			if deployment.Spec.Template.Spec.Containers[0].Image != image {
				deployment.Spec.Template.Spec.Containers[0].Image = image
				updated = true
			}

			if len(deployment.Spec.Template.Spec.Containers[0].Ports) > 0 &&
				deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort != port {
				deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort = port
				updated = true
			}
		}

		if updated {
			if err := r.Update(ctx, &deployment); err != nil {
				logger.Error(err, "unable to update Deployment")
				return ctrl.Result{}, err
			}
			logger.Info("Deployment updated", "name", deployment.Name)
		}
	}

	// 4. Service
	serviceName := fmt.Sprintf("%s-service", bookstore.Spec.Name)
	var service corev1.Service

	err = r.Get(ctx, types.NamespacedName{
		Name:      serviceName,
		Namespace: bookstore.Namespace,
	}, &service)

	if err != nil && apierrors.IsNotFound(err) {
		service = corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: bookstore.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: labels,
				Ports: []corev1.ServicePort{
					{
						Port:       port,
						TargetPort: intstr.FromInt(int(port)),
					},
				},
				Type: corev1.ServiceTypeClusterIP,
			},
		}

		if err := controllerutil.SetControllerReference(&bookstore, &service, r.Scheme); err != nil {
			logger.Error(err, "unable to set owner reference on Service")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, &service); err != nil {
			logger.Error(err, "unable to create Service")
			return ctrl.Result{}, err
		}

		logger.Info("Service created", "name", service.Name)

	} else if err != nil {
		logger.Error(err, "unable to get Service")
		return ctrl.Result{}, err
	}

	// 5. Relire le Deployment pour récupérer readyReplicas
	if err := r.Get(ctx, types.NamespacedName{
		Name:      deploymentName,
		Namespace: bookstore.Namespace,
	}, &deployment); err != nil {
		logger.Error(err, "unable to re-fetch Deployment")
		return ctrl.Result{}, err
	}

	// 6. Mettre à jour le status
	bookstore.Status.ReadyReplicas = deployment.Status.ReadyReplicas
	if deployment.Status.ReadyReplicas == replicas {
		bookstore.Status.Phase = "Ready"
	} else {
		bookstore.Status.Phase = "Creating"
	}

	if err := r.Status().Update(ctx, &bookstore); err != nil {
		logger.Error(err, "unable to update BookStore status")
		return ctrl.Result{}, err
	}

	logger.Info("reconciliation finished",
		"bookstore", bookstore.Name,
		"readyReplicas", bookstore.Status.ReadyReplicas,
		"phase", bookstore.Status.Phase,
	)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BookStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storev1alpha1.BookStore{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
