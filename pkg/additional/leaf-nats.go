package additional

import (
	"fmt"

	"github.com/edgefarm/anck/pkg/nats"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	natsImage                   = "ci4rail/edgefarm-nats:66c57aaf"
	natsNamespace               = "nats"
	defaultDomain               = "DEFAULT_DOMAIN"
	defaultJetstreamStoreageDir = "/store"
	deniedExportTopics          = "local.>"
)

// ApplyLeafNats creates the leaf-nats DaemonSet and necessary namespace and configmap
func ApplyLeafNats(client client.Client) error {
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: natsNamespace,
		},
	}
	err := ApplyOrUpdate(client, &namespace)
	if err != nil {
		return err
	}

	natsServer, err := nats.GetNatsServerInfos()
	if err != nil {
		return err
	}

	defaultSysAccountCredsPath := fmt.Sprintf("%s/%s", credsMountDirectory, sysAccountCredsFile)

	opts := []nats.Option{}
	opts = append(opts, nats.WithRemote(natsServer.Addresses.LeafAddress, defaultSysAccountCredsPath, natsServer.SysAccount.SysPublicKey, []string{deniedExportTopics}, []string{deniedExportTopics}))
	opts = append(opts, nats.WithPidFile("/var/run/nats/nats.pid"))
	opts = append(opts, nats.WithCacheResolver(natsServer.SysAccount.OperatorJWT, natsServer.SysAccount.SysPublicKey, natsServer.SysAccount.SysJWT, "/jwt"))
	opts = append(opts, nats.WithJetstream(defaultJetstreamStoreageDir, defaultDomain))
	detaulfNatsConfig := nats.NewConfig(opts...)

	defaultNatsConfigStr, err := detaulfNatsConfig.ToJSON()
	if err != nil {
		return err
	}

	initialLeafNatsConfigConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "leaf-nats-init-config",
			Namespace: natsNamespace,
		},
		Data: map[string]string{
			"nats.json": defaultNatsConfigStr,
		},
	}

	err = ApplyOrUpdate(client, &initialLeafNatsConfigConfigMap)
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
			sysAccountCredsFile: []byte(natsServer.SysAccount.SysCreds),
		},
	}

	err = ApplyOrUpdate(client, &sysAccountCredsSecret)
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
			Name:      "leaf-nats",
			Namespace: natsNamespace,
			Labels: map[string]string{
				"k8s-app": "leaf-nats",
			},
		},
		Spec: v1.DaemonSetSpec{
			MinReadySeconds: 5,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app": "leaf-nats",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"k8s-app":       "leaf-nats",
						"node-dns.host": "nats",
					},
				},
				Spec: corev1.PodSpec{
					ShareProcessNamespace: &[]bool{true}[0],
					InitContainers: []corev1.Container{
						{
							Name:            "init-nats-config",
							Image:           "alpine:3.15.4",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"/bin/sh",
								"-ce",
								"cp /creds-initial/edgefarm-sys.creds /creds/nats-sidecar.creds && cp /creds-initial/edgefarm-sys.creds /creds/edgefarm-sys.creds && MYDOMAIN=$(cat /host/etc/hostname) && sed -i \"s/DEFAULT_DOMAIN/${MYDOMAIN}/g\" /leaf-nats-init-config/nats.json && cp /leaf-nats-init-config/nats.json /config/nats.json",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "leaf-nats-init-config",
									MountPath: "/leaf-nats-init-config",
									ReadOnly:  false,
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
								{
									Name:      "hostname",
									MountPath: "/host/etc/hostname",
									ReadOnly:  true,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "leaf-nats",
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
									Name:          "jetstream",
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
								{
									Name:      "store",
									MountPath: "/store",
									ReadOnly:  false,
								},
							},
						},
					},
					RestartPolicy: "Always",
					Volumes: []corev1.Volume{
						{
							Name: "leaf-nats-init-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "leaf-nats-init-config",
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
						{
							Name: "store",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/data/nats/jetstream_store",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						{
							Name: "hostname",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/hostname",
									Type: &hostPathFile,
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
