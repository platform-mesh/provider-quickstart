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

	corev1 "k8s.io/api/core/v1"
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	configcontroller "github.com/platform-mesh/provider-quickstart/config/controller"
	configkcp "github.com/platform-mesh/provider-quickstart/config/kcp"
	configprovider "github.com/platform-mesh/provider-quickstart/config/provider"
)

// Bootstrap creates all provider resources from embedded YAML files.
// It bootstraps kcp resources (APIResourceSchema, APIExport), provider
// resources (ProviderMetadata, ContentConfiguration, RBAC), and controller
// resources (ServiceAccount, RBAC, kubeconfig Secret).
func Bootstrap(ctx context.Context, config *rest.Config) error {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	cache := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cache)

	logger := klog.FromContext(ctx)

	// Bootstrap kcp resources (APIResourceSchema, APIExport)
	logger.Info("Bootstrapping kcp resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configkcp.FS); err != nil {
		return fmt.Errorf("failed to bootstrap kcp resources: %w", err)
	}

	// Bootstrap provider resources (ProviderMetadata, ContentConfiguration, RBAC)
	logger.Info("Bootstrapping provider resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configprovider.FS); err != nil {
		return fmt.Errorf("failed to bootstrap provider resources: %w", err)
	}

	// Bootstrap controller resources (ServiceAccount, RBAC)
	logger.Info("Bootstrapping controller resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configcontroller.FS); err != nil {
		return fmt.Errorf("failed to bootstrap controller resources: %w", err)
	}

	// Create kubeconfig secret for controller
	logger.Info("Creating controller kubeconfig secret")
	if err := createControllerKubeconfigSecret(ctx, kubeClient, config); err != nil {
		return fmt.Errorf("failed to create controller kubeconfig secret: %w", err)
	}

	logger.Info("Bootstrap completed successfully")
	return nil
}

// createControllerKubeconfigSecret creates a Secret containing a kubeconfig
// that the controller can use to connect to the workspace from outside.
func createControllerKubeconfigSecret(ctx context.Context, client kubernetes.Interface, config *rest.Config) error {
	logger := klog.FromContext(ctx)

	// Wait for the service account token secret to be populated
	var tokenSecret *corev1.Secret
	err := wait.PollUntilContextTimeout(ctx, time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		secret, err := client.CoreV1().Secrets("default").Get(ctx, "wildwest-controller-token", metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.V(2).Info("waiting for service account token secret to be created")
				return false, nil
			}
			return false, err
		}
		if len(secret.Data["token"]) == 0 {
			logger.V(2).Info("waiting for service account token to be populated")
			return false, nil
		}
		tokenSecret = secret
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for service account token: %w", err)
	}

	token := string(tokenSecret.Data["token"])
	caCert := tokenSecret.Data["ca.crt"]

	// Build kubeconfig pointing to this workspace
	kubeconfig := clientcmdapi.Config{
		Kind:       "Config",
		APIVersion: "v1",
		Clusters: map[string]*clientcmdapi.Cluster{
			"workspace": {
				Server:                   config.Host,
				CertificateAuthorityData: caCert,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"controller": {
				Token: token,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"workspace": {
				Cluster:  "workspace",
				AuthInfo: "controller",
			},
		},
		CurrentContext: "workspace",
	}

	kubeconfigBytes, err := yaml.Marshal(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "wildwest-controller-kubeconfig",
			Namespace: "default",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"kubeconfig": kubeconfigBytes,
		},
	}

	_, err = client.CoreV1().Secrets("default").Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Info("kubeconfig secret already exists, updating")
			existing, err := client.CoreV1().Secrets("default").Get(ctx, secret.Name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get existing secret: %w", err)
			}
			secret.ResourceVersion = existing.ResourceVersion
			if _, err = client.CoreV1().Secrets("default").Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("failed to update secret: %w", err)
			}
			logger.Info("updated kubeconfig secret")
			return nil
		}
		return fmt.Errorf("failed to create secret: %w", err)
	}

	logger.Info("created kubeconfig secret", "name", secret.Name)
	return nil
}

func bootstrapFS(ctx context.Context, dynamicClient dynamic.Interface, mapper meta.RESTMapper, cache discovery.CachedDiscoveryInterface, fs embed.FS) error {
	logger := klog.FromContext(ctx)
	var lastErr error
	attempt := 0
	err := wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		attempt++
		logger.Info("bootstrap attempt", "attempt", attempt)
		if err := createResourcesFromFS(ctx, dynamicClient, mapper, fs); err != nil {
			logger.Info("failed to bootstrap resources, retrying", "attempt", attempt, "error", err)
			lastErr = err
			cache.Invalidate()
			return false, nil
		}
		logger.Info("bootstrap succeeded", "attempt", attempt)
		return true, nil
	})
	if err != nil && lastErr != nil {
		return fmt.Errorf("%w: %v", err, lastErr)
	}
	return err
}

func createResourcesFromFS(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, fs embed.FS) error {
	logger := klog.FromContext(ctx)
	files, err := fs.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read embedded filesystem: %w", err)
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
		logger.Info("processing file", "filename", name)
		if err := createResourceFromFS(ctx, client, mapper, name, fs); err != nil {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func createResourceFromFS(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, filename string, fs embed.FS) error {
	logger := klog.FromContext(ctx)
	raw, err := fs.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", filename, err)
	}

	if len(raw) == 0 {
		logger.Info("skipping empty file", "filename", filename)
		return nil
	}

	d := kubeyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(raw)))
	var errs []error
	for i := 1; ; i++ {
		doc, err := d.Read()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return fmt.Errorf("failed to read YAML document %d from %s: %w", i, filename, err)
		}
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		if err := createResource(ctx, client, mapper, doc); err != nil {
			errs = append(errs, fmt.Errorf("failed to create resource from %s doc %d: %w", filename, i, err))
		}
	}
	return utilerrors.NewAggregate(errs)
}

func createResource(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, raw []byte) error {
	logger := klog.FromContext(ctx)

	u := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(raw, &u.Object); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	gvk := u.GroupVersionKind()
	if gvk.Kind == "" {
		return fmt.Errorf("missing kind in resource")
	}

	logger = logger.WithValues("kind", gvk.Kind, "name", u.GetName(), "namespace", u.GetNamespace())
	logger.Info("resolving REST mapping", "gvk", gvk.String())

	m, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		logger.Error(err, "failed to get REST mapping")
		return fmt.Errorf("failed to get REST mapping for %s: %w", gvk, err)
	}

	logger.Info("creating resource", "resource", m.Resource.String())
	_, err = client.Resource(m.Resource).Namespace(u.GetNamespace()).Create(ctx, u, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Info("resource already exists, updating")
			existing, err := client.Resource(m.Resource).Namespace(u.GetNamespace()).Get(ctx, u.GetName(), metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get existing %s %s: %w", gvk.Kind, u.GetName(), err)
			}

			u.SetResourceVersion(existing.GetResourceVersion())
			if _, err = client.Resource(m.Resource).Namespace(u.GetNamespace()).Update(ctx, u, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("failed to update %s %s: %w", gvk.Kind, u.GetName(), err)
			}
			logger.Info("updated resource")
			return nil
		}
		logger.Error(err, "failed to create resource")
		return fmt.Errorf("failed to create %s %s: %w", gvk.Kind, u.GetName(), err)
	}

	logger.Info("created resource")
	return nil
}
