package additional

import (
	"fmt"

	"github.com/edgefarm/anck/pkg/common"
	"github.com/edgefarm/anck/pkg/nats"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	natsResolverImage          = "nats:2.7.4-alpine3.15"
	credsMountDirectory        = "/creds"
	sysAccountCredsFile        = "edgefarm-sys.creds"
	natsResolverReplicas int32 = 1
)

// ApplyNatsResolver creates the nats-resolver deployment and necessary namespace and configmap
func ApplyNatsResolver(client client.Client) error {
	sys, err := nats.GetSysAccount()
	if err != nil {
		return err
	}

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.AnckNamespace,
		},
	}

	err = ApplyIgnoreExisting(client, &namespace)
	if err != nil {
		return err
	}

	clusterrole := rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "nats-resolver",
			Labels: map[string]string{
				"k8s-app": "nats-resolver",
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
			Name: "nats-resolver",
			Labels: map[string]string{
				"k8s-app": "nats-resolver",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "nats-resolver",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "nats-resolver",
				Namespace: common.AnckNamespace,
			},
		},
	}
	err = ApplyIgnoreExisting(client, &clusterrolebinding)
	if err != nil {
		return err
	}

	serviceaccount := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nats-resolver",
			Namespace: common.AnckNamespace,
			Labels: map[string]string{
				"app": "nats-resolver",
			},
		},
	}
	err = ApplyIgnoreExisting(client, &serviceaccount)
	if err != nil {
		return err
	}

	operatorJWT := sys.OperatorJWT
	sysAccountPubKey := sys.SysAccountPubKey
	sysAccountJWT := sys.SysAccountJWT
	sysAccountCreds := sys.SysAccountCreds
	jwtStoragePath := "/jwt"

	natsResolverConfig := nats.NewConfig(
		nats.WithNGSRemote(fmt.Sprintf("%s/%s", credsMountDirectory, sysAccountCredsFile), sysAccountPubKey),
		nats.WithFullResolver(operatorJWT,
			sysAccountPubKey,
			sysAccountJWT,
			jwtStoragePath))

	natsResolverConfigJSON, err := natsResolverConfig.ToJSON()
	if err != nil {
		return err
	}

	natsResolverConfigmap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nats-resolver-config",
			Namespace: common.AnckNamespace,
		},
		Immutable: func() *bool { b := true; return &b }(),
		Data:      map[string]string{"nats.json": natsResolverConfigJSON},
	}

	err = ApplyIgnoreExisting(client, &natsResolverConfigmap)
	if err != nil {
		return err
	}

	serviceAccountCredsSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nats-resolver-creds",
			Namespace: common.AnckNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			sysAccountCredsFile: []byte(sysAccountCreds),
		},
	}
	err = ApplyIgnoreExisting(client, &serviceAccountCredsSecret)
	if err != nil {
		return err
	}

	deployment := v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nats-resolver",
			Namespace: common.AnckNamespace,
			Labels: map[string]string{
				"k8s-app": "nats-resolver",
			},
		},
		Spec: v1.DeploymentSpec{
			Replicas: &[]int32{natsResolverReplicas}[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": "nats-resolver",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"k8s-app": "nats-resolver",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "nats-resolver",
					Containers: []corev1.Container{
						{
							Name:  "nats-resolver",
							Image: natsResolverImage,
							Args: []string{
								"-js",
								"-c",
								"/config/nats.json",
							},
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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "nats-resolver-config",
									MountPath: "/config",
									ReadOnly:  true,
								},
								{
									Name:      "nats-resolver-creds",
									MountPath: credsMountDirectory,
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "nats-resolver-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "nats-resolver-config",
									},
								},
							},
						},
						{
							Name: "nats-resolver-creds",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "nats-resolver-creds",
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
