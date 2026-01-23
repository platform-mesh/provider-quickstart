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

package wildwest

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	wildwestv1alpha1 "github.com/platform-mesh/provider-quickstart/apis/wildwest/v1alpha1"
)

// CowboyReconciler reconciles a Cowboy object
type CowboyReconciler struct {
	Manager mcmanager.Manager
}

// SetupWithManager sets up the controller with the Manager.
func (r *CowboyReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	r.Manager = mgr

	return mcbuilder.ControllerManagedBy(mgr).
		Named("cowboy-controller").
		For(&wildwestv1alpha1.Cowboy{}).
		Complete(mcreconcile.Func(r.Reconcile))
}

// Reconcile handles reconciliation of Cowboy resources across clusters.
func (r *CowboyReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("cluster", req.ClusterName)

	cl, err := r.Manager.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	client := cl.GetClient()

	// Retrieve the Cowboy from the cluster.
	cowboy := &wildwestv1alpha1.Cowboy{}
	if err := client.Get(ctx, req.NamespacedName, cowboy); err != nil {
		if apierrors.IsNotFound(err) {
			// Cowboy was deleted.
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get cowboy: %w", err)
	}

	log.Info("Reconciling Cowboy", "name", cowboy.Name, "namespace", cowboy.Namespace, "intent", cowboy.Spec.Intent)

	// Update status based on intent
	if cowboy.Spec.Intent != "" && cowboy.Status.Result == "" {
		cowboy.Status.Result = fmt.Sprintf("Yeehaw! %s completed", cowboy.Spec.Intent)
		if err := client.Status().Update(ctx, cowboy); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update cowboy status: %w", err)
		}
		log.Info("Updated Cowboy status", "result", cowboy.Status.Result)
	}

	// Record an event
	recorder := cl.GetEventRecorderFor("cowboy-controller")
	recorder.Eventf(cowboy, corev1.EventTypeNormal, "Reconciled", "Cowboy %s/%s reconciled", cowboy.Namespace, cowboy.Name)

	return reconcile.Result{}, nil
}
