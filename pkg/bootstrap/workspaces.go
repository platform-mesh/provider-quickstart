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

package bootstrap

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// WorkspaceSpec describes a kcp Workspace to create as part of seeding.
type WorkspaceSpec struct {
	// Name of the workspace (e.g. "providers", "quickstart").
	Name string
	// TypeName is the WorkspaceType name (e.g. "providers", "provider").
	TypeName string
	// TypePath is the absolute path of the workspace that owns the type
	// (e.g. "root"). Empty defers to the parent's default behavior.
	TypePath string
}

// ParseWorkspaceSpec parses "<name>=<type-path>:<type-name>" into a WorkspaceSpec.
// "<type-path>:" may be omitted to leave the type path empty.
func ParseWorkspaceSpec(raw string) (WorkspaceSpec, error) {
	name, typeRef, ok := strings.Cut(raw, "=")
	if !ok || name == "" || typeRef == "" {
		return WorkspaceSpec{}, fmt.Errorf("workspace spec %q is not in <name>=<type-path>:<type-name> form", raw)
	}
	spec := WorkspaceSpec{Name: name}
	if path, typeName, ok := strings.Cut(typeRef, ":"); ok {
		spec.TypePath = path
		spec.TypeName = typeName
	} else {
		spec.TypeName = typeRef
	}
	if spec.TypeName == "" {
		return WorkspaceSpec{}, fmt.Errorf("workspace spec %q has empty type name", raw)
	}
	return spec, nil
}

var workspaceGVR = schema.GroupVersionResource{
	Group:    "tenancy.kcp.io",
	Version:  "v1alpha1",
	Resource: "workspaces",
}

// SeedWorkspaces creates the given workspace hierarchy starting from
// parentPath (e.g. "root"). Workspaces are created one nested inside the
// previous one. Existing workspaces are left in place. Returns a rest.Config
// pointing at the deepest (leaf) workspace, suitable for the bootstrap step.
func SeedWorkspaces(ctx context.Context, config *rest.Config, parentPath string, specs []WorkspaceSpec) (*rest.Config, error) {
	if parentPath == "" {
		return nil, fmt.Errorf("parent workspace path must not be empty")
	}
	if len(specs) == 0 {
		return nil, fmt.Errorf("at least one workspace spec is required")
	}

	logger := klog.FromContext(ctx)

	base, err := baseURL(config.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to derive base URL from kubeconfig host: %w", err)
	}

	currentPath := parentPath
	for _, spec := range specs {
		parentConfig := configForCluster(config, base, currentPath)
		logger.Info("Creating workspace", "name", spec.Name, "parent", currentPath, "type", spec.TypeName)
		if err := createWorkspace(ctx, parentConfig, spec); err != nil {
			return nil, fmt.Errorf("failed to create workspace %s in %s: %w", spec.Name, currentPath, err)
		}
		if err := waitForWorkspaceReady(ctx, parentConfig, spec.Name); err != nil {
			return nil, fmt.Errorf("workspace %s in %s did not become ready: %w", spec.Name, currentPath, err)
		}
		currentPath = currentPath + ":" + spec.Name
	}

	return configForCluster(config, base, currentPath), nil
}

// baseURL extracts the scheme://host[:port] from a kcp server URL, dropping
// any /clusters/<path> suffix that may already be present.
func baseURL(host string) (string, error) {
	u, err := url.Parse(host)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("kubeconfig host %q is missing scheme or host", host)
	}
	return (&url.URL{Scheme: u.Scheme, Host: u.Host}).String(), nil
}

// configForCluster returns a copy of config with Host set to the cluster URL
// for the given absolute workspace path.
func configForCluster(config *rest.Config, base, clusterPath string) *rest.Config {
	out := rest.CopyConfig(config)
	out.Host = base + "/clusters/" + clusterPath
	return out
}

func createWorkspace(ctx context.Context, parentConfig *rest.Config, spec WorkspaceSpec) error {
	logger := klog.FromContext(ctx)

	dyn, err := dynamic.NewForConfig(parentConfig)
	if err != nil {
		return fmt.Errorf("failed to build dynamic client: %w", err)
	}

	ws := &unstructured.Unstructured{}
	ws.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   workspaceGVR.Group,
		Version: workspaceGVR.Version,
		Kind:    "Workspace",
	})
	ws.SetName(spec.Name)
	typeRef := map[string]any{"name": spec.TypeName}
	if spec.TypePath != "" {
		typeRef["path"] = spec.TypePath
	}
	if err := unstructured.SetNestedMap(ws.Object, typeRef, "spec", "type"); err != nil {
		return fmt.Errorf("failed to set workspace type: %w", err)
	}

	client := dyn.Resource(workspaceGVR)
	if _, err := client.Create(ctx, ws, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Info("Workspace already exists, leaving as-is", "name", spec.Name)
			return nil
		}
		return err
	}
	logger.Info("Created workspace", "name", spec.Name)
	return nil
}

func waitForWorkspaceReady(ctx context.Context, parentConfig *rest.Config, name string) error {
	logger := klog.FromContext(ctx)

	dyn, err := dynamic.NewForConfig(parentConfig)
	if err != nil {
		return fmt.Errorf("failed to build dynamic client: %w", err)
	}
	client := dyn.Resource(workspaceGVR)

	return wait.PollUntilContextTimeout(ctx, time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		ws, err := client.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		phase, _, _ := unstructured.NestedString(ws.Object, "status", "phase")
		if phase == "Ready" {
			return true, nil
		}
		logger.V(2).Info("Waiting for workspace to be Ready", "name", name, "phase", phase)
		return false, nil
	})
}
