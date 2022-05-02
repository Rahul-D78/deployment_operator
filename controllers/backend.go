package controllers

import (
	"context"
	automation "demo/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func backendDeploymentName(auto *automation.Automation) string {
	return auto.Name + "-backend"
}

func (r *AutomationReconciler) backendDeployment(auto *automation.Automation) *appsv1.Deployment {
	labels := labels(auto, "backend")
	size := auto.Spec.Size
	var ContainerPort int32 = 12
	var backendImage string = "demo:image"

	userSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlAuthName},
			Key:                  "username",
		},
	}

	passwordSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlAuthName},
			Key:                  "password",
		},
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "depl",
			Namespace: auto.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           backendImage,
						ImagePullPolicy: corev1.PullAlways,
						Name:            "visitors-service",
						Ports: []corev1.ContainerPort{{
							ContainerPort: ContainerPort,
							Name:          "visitors",
						}},
						Env: []corev1.EnvVar{
							{
								Name:  "MYSQL_DATABASE",
								Value: "visitors",
							},
							{
								Name:  "MYSQL_SERVICE_HOST",
								Value: mysqlServiceName,
							},
							{
								Name:      "MYSQL_USERNAME",
								ValueFrom: userSecret,
							},
							{
								Name:      "MYSQL_PASSWORD",
								ValueFrom: passwordSecret,
							},
						},
					}},
				},
			},
		},
	}

	controllerutil.SetControllerReference(auto, dep, r.Scheme)
	return dep
}

func (r *AutomationReconciler) handleBackendChanges(auto *automation.Automation) (*reconcile.Result, error) {

	foundDepl := &appsv1.Deployment{}
	deplKey := types.NamespacedName{Name: backendDeploymentName(auto), Namespace: auto.Namespace}

	err := r.Client.Get(context.TODO(), deplKey, foundDepl)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Error(err, "Not found may be object got deleted requeing")
			return &reconcile.Result{Requeue: true}, err
		}
	}

	return &reconcile.Result{}, err
}
