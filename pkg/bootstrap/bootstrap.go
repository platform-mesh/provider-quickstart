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
	"bufio"
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	configkcp "github.com/platform-mesh/provider-quickstart/config/kcp"
	configprovider "github.com/platform-mesh/provider-quickstart/config/provider"
)

// Bootstrap creates all provider resources from embedded YAML files.
// It bootstraps KCP resources (APIResourceSchema, APIExport) and provider
// resources (ProviderMetadata, ContentConfiguration, RBAC).
func Bootstrap(ctx context.Context, config *rest.Config) error {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	cache := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cache)

	logger := klog.FromContext(ctx)

	// Bootstrap KCP resources (APIResourceSchema, APIExport)
	logger.Info("Bootstrapping KCP resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configkcp.FS); err != nil {
		return fmt.Errorf("failed to bootstrap KCP resources: %w", err)
	}

	// Bootstrap provider resources (ProviderMetadata, ContentConfiguration, RBAC)
	logger.Info("Bootstrapping provider resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configprovider.FS); err != nil {
		return fmt.Errorf("failed to bootstrap provider resources: %w", err)
	}

	logger.Info("Bootstrap completed successfully")
	return nil
}

func bootstrapFS(ctx context.Context, dynamicClient dynamic.Interface, mapper meta.RESTMapper, cache discovery.CachedDiscoveryInterface, fs embed.FS) error {
	return wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		if err := createResourcesFromFS(ctx, dynamicClient, mapper, fs); err != nil {
			klog.FromContext(ctx).V(2).Info("failed to bootstrap resources, retrying", "err", err)
			cache.Invalidate()
			return false, nil
		}
		return true, nil
	})
}

func createResourcesFromFS(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, fs embed.FS) error {
	files, err := fs.ReadDir(".")
	if err != nil {
		return err
	}

	var errs []error
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		// Skip non-yaml files (like bootstrap.go)
		name := f.Name()
		if len(name) < 5 || (name[len(name)-5:] != ".yaml" && name[len(name)-4:] != ".yml") {
			continue
		}
		if err := createResourceFromFS(ctx, client, mapper, name, fs); err != nil {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func createResourceFromFS(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, filename string, fs embed.FS) error {
	raw, err := fs.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", filename, err)
	}

	if len(raw) == 0 {
		return nil
	}

	d := kubeyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(raw)))
	var errs []error
	for i := 1; ; i++ {
		doc, err := d.Read()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		if err := createResource(ctx, client, mapper, doc); err != nil {
			errs = append(errs, fmt.Errorf("failed to create resource %s doc %d: %w", filename, i, err))
		}
	}
	return utilerrors.NewAggregate(errs)
}

func createResource(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, raw []byte) error {
	logger := klog.FromContext(ctx)

	u := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(raw, &u.Object); err != nil {
		return fmt.Errorf("could not decode raw: %w", err)
	}

	gvk := u.GroupVersionKind()
	if gvk.Kind == "" {
		return fmt.Errorf("missing kind in resource")
	}

	m, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("could not get REST mapping for %s: %w", gvk, err)
	}

	logger = logger.WithValues("kind", gvk.Kind, "name", u.GetName())

	_, err = client.Resource(m.Resource).Namespace(u.GetNamespace()).Create(ctx, u, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			existing, err := client.Resource(m.Resource).Namespace(u.GetNamespace()).Get(ctx, u.GetName(), metav1.GetOptions{})
			if err != nil {
				return err
			}

			u.SetResourceVersion(existing.GetResourceVersion())
			if _, err = client.Resource(m.Resource).Namespace(u.GetNamespace()).Update(ctx, u, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("could not update %s %s: %w", gvk.Kind, u.GetName(), err)
			}
			logger.Info("updated resource")
			return nil
		}
		return err
	}

	logger.Info("created resource")
	return nil
}
