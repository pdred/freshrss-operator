/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	freshrssv1alpha1 "github.com/saas-patterns/freshrss-operator/api/v1alpha1"
)

// FreshRSSReconciler reconciles a FreshRSS object
type FreshRSSReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=freshrss.demo.openshift.com,resources=freshrsses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=freshrss.demo.openshift.com,resources=freshrsses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=freshrss.demo.openshift.com,resources=freshrsses/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete

func (r *FreshRSSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	instance := &freshrssv1alpha1.FreshRSS{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	for _, component := range []component{
		{"Deployment", "", r.newDeployment},
		{"Service", "", r.newService},
		{"Route", "", r.newRoute},
	} {
		obj, mutateFn, err := component.fn(ctx, instance)
		if err != nil {
			log.Error(err, "Failed to mutate resource", "Kind", component.name)
			return ctrl.Result{}, err
		}

		result, err := controllerutil.CreateOrUpdate(ctx, r.Client, obj, mutateFn)
		if err != nil {
			log.Error(err, "Failed to create or update", "Kind", component.name)
			return ctrl.Result{}, err
		}
		switch result {
		case controllerutil.OperationResultCreated:
			log.Info("Created " + component.name)
			return ctrl.Result{}, nil
		case controllerutil.OperationResultUpdated:
			log.Info("Updated " + component.name)
			return ctrl.Result{}, nil
		}
	}

	route := &routev1.Route{}
	err = r.Client.Get(ctx, req.NamespacedName, route)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	}

	if len(route.Status.Ingress) == 1 {
		instance.Status.URL = route.Status.Ingress[0].Host
		if route.Status.Ingress[0].Host == "" {
			instance.Status.URL = ""
		} else {
			instance.Status.URL = "http://" + route.Status.Ingress[0].Host
		}
	}

	return ctrl.Result{}, r.Status().Update(ctx, instance)
}

func (r *FreshRSSReconciler) newRoute(ctx context.Context, instance *freshrssv1alpha1.FreshRSS) (client.Object, controllerutil.MutateFn, error) {
	weight := int32(100)
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
	}
	routeSpec := routev1.RouteSpec{
		To: routev1.RouteTargetReference{
			Kind:   "Service",
			Name:   instance.Name,
			Weight: &weight,
		},
		Port: &routev1.RoutePort{
			TargetPort: intstr.FromInt(8080),
		},
		WildcardPolicy: routev1.WildcardPolicyNone,
	}

	mutateFn := func() error {
		if err := controllerutil.SetControllerReference(instance, route, r.Scheme); err != nil {
			return err
		}

		// don't clobber other values set in the route spec
		route.Spec.To = routeSpec.To
		route.Spec.Port = routeSpec.Port
		route.Spec.WildcardPolicy = routeSpec.WildcardPolicy
		return nil
	}

	return route, mutateFn, nil

}

func (r *FreshRSSReconciler) newService(ctx context.Context, instance *freshrssv1alpha1.FreshRSS) (client.Object, controllerutil.MutateFn, error) {
	labels := map[string]string{
		"app": instance.Name,
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
	}

	mutateFn := func() error {
		if err := controllerutil.SetControllerReference(instance, service, r.Scheme); err != nil {
			return err
		}

		service.Spec.Ports = []corev1.ServicePort{
			{
				Port:       80,
				TargetPort: intstr.FromInt(8080),
				Protocol:   corev1.ProtocolTCP,
			},
		}
		service.Spec.Selector = labels

		return nil
	}

	return service, mutateFn, nil
}

func (r *FreshRSSReconciler) newDeployment(ctx context.Context, instance *freshrssv1alpha1.FreshRSS) (client.Object, controllerutil.MutateFn, error) {
	container := corev1.Container{
		Name:  "freshrss",
		Image: "quay.io/saas-patterns/freshrss-image:latest",
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: int32(8080),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		ImagePullPolicy: corev1.PullAlways,
		Env: []corev1.EnvVar{
			{
				Name:  "TITLE",
				Value: instance.Spec.Title,
			},
			{
				Name:  "DEFAULTUSER",
				Value: instance.Spec.DefaultUser,
			},
		},
	}

	labels := map[string]string{
		"app": instance.ObjectMeta.Name,
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Name:   instance.ObjectMeta.Name,
				},
			},
		},
	}

	mutateFn := func() error {
		if err := controllerutil.SetControllerReference(instance, deployment, r.Scheme); err != nil {
			return err
		}
		var replicas int32 = 1
		deployment.Spec.Replicas = &replicas

		// don't clobber fields that were defaulted
		if len(deployment.Spec.Template.Spec.Containers) != 1 {
			deployment.Spec.Template.Spec.Containers = []corev1.Container{container}
		} else {
			c := deployment.Spec.Template.Spec.Containers[0]
			c.Name = container.Name
			c.Image = container.Image
			c.Ports = container.Ports
			c.ImagePullPolicy = container.ImagePullPolicy
			c.Env = container.Env
		}

		return nil
	}
	return deployment, mutateFn, nil
}

type NewComponentFn func(context.Context, *freshrssv1alpha1.FreshRSS) (client.Object, controllerutil.MutateFn, error)

type component struct {
	name   string
	reason string
	fn     NewComponentFn
}

// SetupWithManager sets up the controller with the Manager.
func (r *FreshRSSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&freshrssv1alpha1.FreshRSS{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&routev1.Route{}).
		Complete(r)
}
