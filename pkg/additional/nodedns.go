package additional

import (
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nodeDNSImage     = "ghcr.io/edgefarm/node-dns:1.0.0-beta.2"
	nodeDNSNamespace = "node-dns"
)

// ApplyNodeDNS creates the NodeDNS DaemonSet and necessary namespace and configmap
func ApplyNodeDNS(client client.Client) error {
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeDNSNamespace,
		},
	}
	err := ApplyOrUpdate(client, &namespace)
	if err != nil {
		return err
	}

	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "node-dns-cfg",
			Namespace: nodeDNSNamespace,
		},
		Data: map[string]string{
			"node-dns.conf": `listeninterface: docker0
listenport: 53
updateresolvconf: true
resolvConf: /etc/resolv.conf
removeSearchDomains: true
feed:
  k8sapi:
    enabled: true
    insecuretls: true
    token: ""
    uri: http://127.0.0.1:10550
`,
		},
	}

	err = ApplyOrUpdate(client, &configMap)
	if err != nil {
		return err
	}

	daemonset := v1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "node-dns",
			Namespace: nodeDNSNamespace,
			Labels: map[string]string{
				"k8s-app": "node-dns",
			},
		},
		Spec: v1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": "node-dns",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"k8s-app": "node-dns",
					},
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					Containers: []corev1.Container{
						{
							Name: "node-dns",
							SecurityContext: &corev1.SecurityContext{
								Privileged: &[]bool{true}[0],
							},
							Image:   nodeDNSImage,
							Command: []string{"/node-dns"},
							Args: []string{
								"--config",
								"/config/node-dns.conf",
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("50m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "conf",
									MountPath: "/config",
									ReadOnly:  true,
								},
								{
									Name:      "resolv",
									ReadOnly:  false,
									MountPath: "/etc/resolv.conf",
								},
							},
						},
					},
					RestartPolicy: "Always",
					Volumes: []corev1.Volume{
						{
							Name: "conf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "node-dns-cfg",
									},
								},
							},
						},
						{
							Name: "resolv",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/resolv.conf",
								},
							},
						},
					},
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "node-role.kubernetes.io/edge",
												Operator: "Exists",
											},
											{
												Key:      "node-role.kubernetes.io/agent",
												Operator: "Exists",
											},
										},
									},
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      "edgefarm.applications",
							Operator: "Exists",
							Effect:   "NoExecute",
						},
					},
				},
			},
		},
	}
	err = ApplyOrUpdate(client, &daemonset)
	if err != nil {
		return err
	}

	return nil
}
