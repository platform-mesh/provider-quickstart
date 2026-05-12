/*
Copyright 2025 The Platform Mesh Authors.

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
	"os"
	"time"

	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	wildwestv1alpha1 "github.com/platform-mesh/provider-quickstart/apis/wildwest/v1alpha1"
	armamentsync "github.com/platform-mesh/provider-quickstart/operator/armament-sync"
	"github.com/platform-mesh/provider-quickstart/pkg/external/static"
)

func init() {
	runtime.Must(wildwestv1alpha1.AddToScheme(scheme.Scheme))
}

func main() {
	log.SetLogger(zap.New(zap.UseDevMode(true)))

	ctx := signals.SetupSignalHandler()
	entryLog := log.Log.WithName("entrypoint")

	var syncInterval time.Duration
	pflag.DurationVar(&syncInterval, "sync-interval", 30*time.Second, "How often to reconcile the armament catalog against the external source")
	pflag.Parse()

	cfg := ctrl.GetConfigOrDie()

	mgr, err := manager.New(cfg, manager.Options{
		Scheme:                 scheme.Scheme,
		HealthProbeBindAddress: ":8081",
		Metrics: metricsserver.Options{
			BindAddress: ":9081",
		},
	})
	if err != nil {
		entryLog.Error(err, "unable to set up manager")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		entryLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		entryLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	syncer := &armamentsync.Syncer{
		Client:   mgr.GetClient(),
		Source:   static.New(),
		Interval: syncInterval,
	}
	if err := syncer.AddToManager(mgr); err != nil {
		entryLog.Error(err, "unable to add armament syncer")
		os.Exit(1)
	}

	entryLog.Info("Starting armament-sync manager", "interval", syncInterval)
	if err := mgr.Start(ctx); err != nil {
		entryLog.Error(err, "manager exited with error")
		os.Exit(1)
	}
}
