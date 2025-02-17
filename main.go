// Copyright 2022 Red Hat, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"os"

	"k8s.io/klog/v2/klogr"

	"k8s.io/klog/v2"

	"github.com/kiegroup/kogito-serverless-operator/utils"

	"github.com/kiegroup/kogito-serverless-operator/controllers"
	ocputil "github.com/kiegroup/kogito-serverless-operator/utils/openshift"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/log"
	//+kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorapi.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	klog.InitFlags(nil)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(klogr.New().WithName(controllers.ComponentName))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "1be5e57d.kie.org",
	})
	if err != nil {
		klog.V(log.E).ErrorS(err, "unable to start manager")
		os.Exit(1)
	}

	utils.SetIsOpenShift(mgr.GetConfig())

	if err = (&controllers.SonataFlowReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Config:   mgr.GetConfig(),
		Recorder: mgr.GetEventRecorderFor("workflow-controller"),
	}).SetupWithManager(mgr); err != nil {
		klog.V(log.E).ErrorS(err, "unable to create controller", "controller", "SonataFlow")
		os.Exit(1)
	}
	if err = (&controllers.SonataFlowBuildReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Config:   mgr.GetConfig(),
		Recorder: mgr.GetEventRecorderFor("build-controller"),
	}).SetupWithManager(mgr); err != nil {
		klog.V(log.E).ErrorS(err, "unable to create controller", "controller", "SonataFlowBuild")
		os.Exit(1)
	}

	if err = (&controllers.SonataFlowPlatformReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Reader:   mgr.GetAPIReader(),
		Config:   mgr.GetConfig(),
		Recorder: mgr.GetEventRecorderFor("platform-controller"),
	}).SetupWithManager(mgr); err != nil {
		klog.V(log.E).ErrorS(err, "unable to create controller", "controller", "SonataFlowPlatform")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if utils.IsOpenShift() {
		ocputil.MustAddToScheme(mgr.GetScheme())
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		klog.V(log.E).ErrorS(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		klog.V(log.E).ErrorS(err, "unable to set up ready check")
		os.Exit(1)
	}

	klog.V(log.I).InfoS("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.V(log.E).ErrorS(err, "problem running manager")
		os.Exit(1)
	}

}
