package revision

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceAccountOverrides func(sa *corev1.ServiceAccount)

func ServiceAccountWithAdditionalPullSecrets(secrets []corev1.LocalObjectReference) ServiceAccountOverrides {
	return func(sa *corev1.ServiceAccount) {
		sa.ImagePullSecrets = append(sa.ImagePullSecrets, secrets...)
	}
}

func ServiceAccountWithNamespace(namespace string) ServiceAccountOverrides {
	return func(sa *corev1.ServiceAccount) {
		sa.Namespace = namespace
	}
}

func ServiceAccountWithOwnerReferences(owners []metav1.OwnerReference) ServiceAccountOverrides {
	return func(sa *corev1.ServiceAccount) {
		sa.OwnerReferences = owners
	}
}

type DeploymentOverrides func(deployment *appsv1.Deployment)

func DeploymentWithNamespace(namespace string) DeploymentOverrides {
	return func(d *appsv1.Deployment) {
		d.Namespace = namespace
	}
}

func DeploymentWithOwnerReferences(owners []metav1.OwnerReference) DeploymentOverrides {
	return func(d *appsv1.Deployment) {
		d.OwnerReferences = owners
	}
}

func DeploymentWithSelectors(selectors map[string]string) DeploymentOverrides {
	return func(d *appsv1.Deployment) {
		d.Spec.Selector.MatchLabels = selectors
		// Ensure that the pod template labels always contains the deployment
		// selector.
		for k, v := range selectors {
			d.Spec.Template.Labels[k] = v
		}
	}
}

func DeploymentWithServiceAccount(sa string) DeploymentOverrides {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.ServiceAccountName = sa
	}
}

func DeploymentWithImagePullSecrets(secrets []corev1.LocalObjectReference) DeploymentOverrides {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.ImagePullSecrets = secrets
	}
}

func DeploymentRuntimeWithImage(image string) DeploymentOverrides {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Containers[0].Image = image
	}
}

func DeploymentRuntimeWithImagePullPolicy(policy corev1.PullPolicy) DeploymentOverrides {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Containers[0].ImagePullPolicy = policy
	}
}

func DeploymentRuntimeWithTLSServerSecret(secret string) DeploymentOverrides {
	return func(d *appsv1.Deployment) {
		mountTLSSecret(secret, tlsServerCertsVolumeName, tlsServerCertsDir, tlsServerCertDirEnvVar, d)
	}
}

func DeploymentRuntimeWithTLSClientSecret(secret string) DeploymentOverrides {
	return func(d *appsv1.Deployment) {
		mountTLSSecret(secret, tlsClientCertsVolumeName, tlsClientCertsDir, tlsClientCertDirEnvVar, d)
	}
}

func DeploymentRuntimeWithAdditionalEnvironments(env []corev1.EnvVar) DeploymentOverrides {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Containers[0].Env = append(d.Spec.Template.Spec.Containers[0].Env, env...)
	}
}

func DeploymentRuntimeWithAdditionalPorts(ports []corev1.ContainerPort) DeploymentOverrides {
	return func(d *appsv1.Deployment) {
		d.Spec.Template.Spec.Containers[0].Ports = append(d.Spec.Template.Spec.Containers[0].Ports, ports...)
	}
}
