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
	"context"
	"os"
	"time"

	"github.com/spf13/pflag"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/platform-mesh/provider-quickstart/pkg/bootstrap"
)

func main() {
	klog.InitFlags(nil)

	var kubeconfig string
	pflag.StringVar(&kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to kubeconfig file")
	pflag.Parse()

	if kubeconfig == "" {
		klog.Fatal("--kubeconfig is required or set KUBECONFIG environment variable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	logger := klog.FromContext(ctx)

	logger.Info("Loading kubeconfig", "path", kubeconfig)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		klog.Fatal("Failed to build config from kubeconfig", "err", err)
	}

	logger.Info("Bootstrapping provider resources")

	if err := bootstrap.Bootstrap(ctx, config); err != nil {
		klog.Fatal("Failed to bootstrap", "err", err)
	}

	logger.Info("Bootstrap completed successfully")
}
