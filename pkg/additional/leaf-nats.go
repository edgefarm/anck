package additional

import (
	"fmt"

	"github.com/edgefarm/anck/pkg/nats"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	natsImage     = "ci4rail/edgefarm-nats:796af8fb"
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

	sys, err := nats.GetSysAccount()
	if err != nil {
		return err
	}

	defaultSysAccountCredsPath := fmt.Sprintf("%s/%s", credsMountDirectory, sysAccountCredsFile)

	opts := []nats.Option{}
	opts = append(opts, nats.WithNGSRemote(defaultSysAccountCredsPath, sys.SysAccountPubKey))
	opts = append(opts, nats.WithPidFile("/var/run/nats/nats.pid"))
	opts = append(opts, nats.WithCacheResolver(sys.OperatorJWT, sys.SysAccountPubKey, sys.SysAccountJWT, "/jwt"))
	detaulfNatsConfig := nats.NewConfig(opts...)

	defaultNatsConfigStr, err := detaulfNatsConfig.ToJSON()
	if err != nil {
		return err
	}

	initialNatsConfigConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nats-init-config",
			Namespace: natsNamespace,
		},
		Data: map[string]string{
			"nats.json": defaultNatsConfigStr,
		},
	}

	err = ApplyOrUpdate(client, &initialNatsConfigConfigMap)
	if err != nil {
		return err
	}

	sysAccountCredsSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "initial-nats-sys-account-creds",
			Namespace: natsNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			sysAccountCredsFile: []byte(sys.SysAccountCreds),
		},
	}

	err = ApplyOrUpdate(client, &sysAccountCredsSecret)
	if err != nil {
		return err
	}

	hostPathDirectoryOrCreate := corev1.HostPathDirectoryOrCreate

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
								"cp /creds-initial/edgefarm-sys.creds /creds/nats-sidecar.creds && cp /creds-initial/edgefarm-sys.creds /creds/edgefarm-sys.creds && [ ! -f /config/nats.json ] && echo `cat /nats-init-config/nats.json` > /config/nats.json || echo nats.json already exists",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "nats-init-config",
									MountPath: "/nats-init-config",
									ReadOnly:  true,
								},
								{
									Name:      "initial-nats-sys-account-creds",
									MountPath: "/creds-initial",
									ReadOnly:  true,
								},
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
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "nats",
							Image:           natsImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
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
									Name:      "state",
									MountPath: "/state",
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
							Name: "initial-nats-sys-account-creds",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "initial-nats-sys-account-creds",
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
