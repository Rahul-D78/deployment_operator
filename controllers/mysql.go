package controllers

import (
	automation "demo/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	mysqlAuthName    = "mysql-auth"
	mySqlDeplName    = "mysql-depl"
	mysqlServiceName = "mysql-svc"
)

func (r *AutomationReconciler) mySqlAuthSecret(v *automation.Automation) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysqlAuthName,
			Namespace: v.Namespace,
		},
		Type: "Opaque",
		StringData: map[string]string{
			"username": "mysql-user",
			"password": "mysql-pass",
		},
	}
	controllerutil.SetControllerReference(v, secret, r.Scheme)
	return secret
}

func (r *AutomationReconciler) deployMySql(v *automation.Automation) *appsv1.Deployment {
	labels := labels(v, "mysql")
	size := int32(1)

	userSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: mysqlAuthName,
			},
			Key: "username",
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
			Name:      mySqlDeplName,
			Namespace: v.Namespace,
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
						Image: "mysql:latest",
						Name:  "auto-mysql",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 3306,
							Name:          "mysql",
						}},
						Env: []corev1.EnvVar{
							{
								Name:  "MYSQL_ROOT_PASSWORD",
								Value: "password",
							},
							{
								Name:  "MYSQL_DATABASE",
								Value: "visitors",
							},
							{
								Name:      "MYSQL_USER",
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

	controllerutil.SetControllerReference(v, dep, r.Scheme)
	return dep
}
