package additional

import (
	"github.com/edgefarm/anck/pkg/common"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	anckcredentialsImage    = "ci4rail/anck-credentials:latest"
	anckcredentialsGrpcPort = 6000
)

// ApplyAnckCredentials creates the nats DaemonSet and necessary namespace and configmap
func ApplyAnckCredentials(client client.Client) error {
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.AnckNamespace,
		},
	}
	err := ApplyIgnoreExisting(client, &namespace)
	if err != nil {
		return err
	}

	clusterrole := rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "anck-credentials",
			Labels: map[string]string{
				"k8s-app": "anck-credentials",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "watch", "create", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get", "list", "create"},
			},
		},
	}
	err = ApplyIgnoreExisting(client, &clusterrole)
	if err != nil {
		return err
	}

	clusterrolebinding := rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "anck-credentials",
			Labels: map[string]string{
				"k8s-app": "anck-credentials",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "anck-credentials",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "anck-credentials",
				Namespace: common.AnckNamespace,
			},
		},
	}
	err = ApplyIgnoreExisting(client, &clusterrolebinding)
	if err != nil {
		return err
	}

	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "anck-credentials",
			Namespace: common.AnckNamespace,
			Labels: map[string]string{
				"app": "anck-credentials",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "grpc",
					Protocol:   "TCP",
					Port:       anckcredentialsGrpcPort,
					TargetPort: intstr.FromInt(anckcredentialsGrpcPort),
				},
			},
			Selector: map[string]string{
				"app": "anck-credentials",
			},
		},
	}
	err = ApplyIgnoreExisting(client, &service)
	if err != nil {
		return err
	}

	serviceaccount := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "anck-credentials",
			Namespace: common.AnckNamespace,
			Labels: map[string]string{
				"app": "anck-credentials",
			},
		},
	}
	err = ApplyIgnoreExisting(client, &serviceaccount)
	if err != nil {
		return err
	}

	deployment := v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "anck-credentials",
			Namespace: common.AnckNamespace,
			Labels: map[string]string{
				"k8s-app": "anck-credentials",
			},
		},
		Spec: v1.DeploymentSpec{
			Replicas: &[]int32{1}[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": "anck-credentials",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"k8s-app": "anck-credentials",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "anck-credentials",
					Containers: []corev1.Container{
						{
							Name:  "anck-credentials",
							Image: anckcredentialsImage,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("250m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	err = ApplyIgnoreExisting(client, &deployment)
	if err != nil {
		return err
	}

	return nil
}
