package additional

import (
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	natsImage     = "nats:2.6.5-alpine3.14"
	natsNamespace = "nats"
)

// ApplyNats creates the nats DaemonSet and necessary namespace and configmap
func ApplyNats(client client.Client) error {
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: natsNamespace,
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
			Name:      "nats-init-config",
			Namespace: natsNamespace,
		},
		Data: map[string]string{
			"nats.json": `{
	"http": 8222,
	"leafnodes": {
		"remotes": []
	},
	"pid_file": "/var/run/nats/nats.pid"
}`,
		},
	}

	err = ApplyOrUpdate(client, &configMap)
	if err != nil {
		return err
	}

	hostPathDirectoryOrCreate := corev1.HostPathDirectoryOrCreate
	hostPathFile := corev1.HostPathFile

	daemonset := v1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nats",
			Namespace: natsNamespace,
			Labels: map[string]string{
				"k8s-app": "nats",
			},
		},
		Spec: v1.DaemonSetSpec{
			MinReadySeconds: 5,
			UpdateStrategy: v1.DaemonSetUpdateStrategy{
				Type: "RollingUpdate",
				RollingUpdate: &v1.RollingUpdateDaemonSet{
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 50,
					},
					MaxSurge: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 50,
					},
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": "nats",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"k8s-app":       "nats",
						"node-dns.host": "nats",
					},
				},
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &[]bool{true}[0],
					InitContainers: []corev1.Container{
						{
							Name:            "init-nats-config",
							Image:           "busybox:1.28",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"/bin/sh",
								"-c",
								"[ ! -f /config/nats.json ] && echo `cat /nats-init-config/nats.json` > /config/nats.json || echo nats.json already exists",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "nats-init-config",
									MountPath: "/nats-init-config",
								},
								{
									Name:      "config",
									MountPath: "/config",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "nats",
							Image:           natsImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args: []string{
								"-js",
								"-c",
								"/config/nats.json",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "clients",
									ContainerPort: 4222,
									HostPort:      4222,
								},
								{
									Name:          "monitor",
									ContainerPort: 8222,
									HostPort:      8222,
								},
								{
									Name:          "jestream",
									ContainerPort: 7222,
									HostPort:      7222,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/config",
									ReadOnly:  false,
								},
								{
									Name:      "creds",
									MountPath: "/creds",
									ReadOnly:  false,
								},
								{
									Name:      "nats-pid",
									MountPath: "/var/run/nats",
									ReadOnly:  false,
								},
							},
						},
						{
							Name:    "nats-leafnode-registry",
							Image:   "ci4rail/nats-leafnode-registry:46c637d4",
							Command: []string{"/registry"},
							Args:    []string{"--natsuri", "nats://localhost:4222"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/config",
									ReadOnly:  false,
								},
								{
									Name:      "creds",
									MountPath: "/creds",
									ReadOnly:  false,
								},
								{
									Name:      "resolv",
									MountPath: "/etc/resolv.conf",
									ReadOnly:  false,
								},
								{
									Name:      "state",
									MountPath: "/state",
									ReadOnly:  false,
								},
							},
						},
						{
							Name:            "nats-config-reloader",
							Image:           "ci4rail/nats-server-config-reloader:7fc8210",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"nats-server-config-reloader"},
							Args: []string{
								"-P", "/var/run/nats/nats.pid",
								"-c", "/config/nats.json",
								"-signal", "15"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/config",
									ReadOnly:  false,
								},
								{
									Name:      "nats-pid",
									MountPath: "/var/run/nats/",
									ReadOnly:  false,
								},
							},
						},
					},
					RestartPolicy: "Always",
					Volumes: []corev1.Volume{
						{
							Name: "nats-init-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "nats-init-config",
									},
								},
							},
						},
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/data/nats/config",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						{
							Name: "creds",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/data/nats/creds",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						{
							Name: "nats-pid",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/data/nats/pid/",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						{
							Name: "resolv",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/resolv.conf",
									Type: &hostPathFile,
								},
							},
						},
						{
							Name: "state",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/data/nats/registry-state",
									Type: &hostPathDirectoryOrCreate,
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
