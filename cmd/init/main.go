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

	var (
		kubeconfig      string
		hostOverride    string
		seedWorkspaces  bool
		parentWorkspace string
		workspaceSpecs  []string
	)

	pflag.StringVar(&kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to kubeconfig file")
	pflag.StringVar(&hostOverride, "host-override", os.Getenv("HOST_OVERRIDE"), "Override the server URL in the generated controller kubeconfig (e.g. https://frontproxy-front-proxy.platform-mesh-system:6443)")
	pflag.BoolVar(&seedWorkspaces, "seed-workspaces", false, "Create the provider workspace hierarchy from the kubeconfig before bootstrapping. Requires an admin kubeconfig pointing at the kcp front-proxy.")
	pflag.StringVar(&parentWorkspace, "parent-workspace", "root", "Absolute path of the parent workspace under which --workspace entries are created (only used with --seed-workspaces).")
	pflag.StringSliceVar(&workspaceSpecs, "workspace", []string{"providers=root:providers", "quickstart=root:provider"}, "Workspace to create when --seed-workspaces is set, formatted as <name>=<type-path>:<type-name>. Repeat (or comma-separate) for nested workspaces in parent-first order. The final entry is the workspace bootstrapped into.")
	pflag.Parse()

	if kubeconfig == "" {
		klog.Fatal("--kubeconfig is required or set KUBECONFIG environment variable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger := klog.FromContext(ctx)

	logger.Info("Loading kubeconfig", "path", kubeconfig)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		klog.Fatal("Failed to build config from kubeconfig", "err", err)
	}

	if seedWorkspaces {
		specs := make([]bootstrap.WorkspaceSpec, 0, len(workspaceSpecs))
		for _, raw := range workspaceSpecs {
			spec, err := bootstrap.ParseWorkspaceSpec(raw)
			if err != nil {
				klog.Fatal(err)
			}
			specs = append(specs, spec)
		}
		logger.Info("Seeding provider workspace hierarchy", "parent", parentWorkspace, "workspaces", workspaceSpecs)
		leaf, err := bootstrap.SeedWorkspaces(ctx, config, parentWorkspace, specs)
		if err != nil {
			klog.Fatal("Failed to seed workspaces", "err", err)
		}
		config = leaf
	}

	logger.Info("Bootstrapping provider resources", "host", config.Host)

	if err := bootstrap.Bootstrap(ctx, config, hostOverride); err != nil {
		klog.Fatal("Failed to bootstrap", "err", err)
	}

	logger.Info("Bootstrap completed successfully")
}
