/*
Copyright © 2021 Ci4Rail GmbH <engineering@ci4rail.com>
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
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	v1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	networkv1alpha1 "github.com/edgefarm/anck/apis/network/v1alpha1"
	networkcontrollers "github.com/edgefarm/anck/controllers/network"
	"github.com/edgefarm/anck/pkg/additional"
	"github.com/edgefarm/anck/pkg/common"
	"github.com/edgefarm/anck/pkg/resources"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(networkv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "3fdd68dd.network.edgefarm.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	setupLog.Info("applying Deployment for 'anck-credentials'")
	err = additional.ApplyAnckCredentials(mgr.GetClient())
	if err != nil {
		setupLog.Error(err, "unable to set up acnk-credentials")
		os.Exit(1)
	}

	setupLog.Info("waiting for 'anck-credentials' to be up and running")
	err = waitForAnckCredentials(1 * time.Minute)
	if err != nil {
		setupLog.Error(err, "anck-credentials timed out. Exiting...")
		os.Exit(1)
	}

	if err = (&networkcontrollers.NetworksReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Network")
		os.Exit(1)
	}
	if err = (&networkcontrollers.ParticipantsReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Participants")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("applying DaemonSet for 'node-dns'")
	err = additional.ApplyNodeDNS(mgr.GetClient())
	if err != nil {
		setupLog.Error(err, "unable to set up node dns")
		os.Exit(1)
	}

	setupLog.Info("applying DaemonSet for 'nats'")
	err = additional.ApplyNats(mgr.GetClient())
	if err != nil {
		setupLog.Error(err, "unable to set up nats")
		os.Exit(1)
	}

	err = additional.ApplyNatsResolver(mgr.GetClient())
	if err != nil {
		setupLog.Error(err, "unable to set up nats-resolver")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

type response struct {
	Err error
}

func waitForAnckCredentials(timeout time.Duration) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan response, 1)

	go func() {
		for {
			select {
			default:
				podList, err := resources.ListPods(common.AnckNamespace)
				if err != nil {
					ch <- response{
						Err: err,
					}
					return
				}
				for _, pod := range podList {
					if strings.Contains(pod.Name, "anck-credentials") {
						if pod.Status.Phase == v1.PodRunning {
							ch <- response{
								Err: nil,
							}
							return
						}
					}
				}
			case <-ctx.Done():
				fmt.Println("Canceled by timeout")
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()

	select {
	case response := <-ch:
		return response.Err
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for anck-credentials")
	}
}
