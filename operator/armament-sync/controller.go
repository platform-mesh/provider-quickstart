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

// Package armamentsync runs in the provider workspace and projects the
// external armament catalog onto Armament custom resources. The CRs are
// then exposed to consumer workspaces read-only via a CachedResource bound
// to the wildwest APIExport.
package armamentsync

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	wildwestv1alpha1 "github.com/platform-mesh/provider-quickstart/apis/wildwest/v1alpha1"
	"github.com/platform-mesh/provider-quickstart/pkg/external"
)

// managedByLabel marks armaments owned by this sync loop so we never
// delete CRs created out-of-band (e.g. by an operator hand-editing).
const managedByLabel = "wildwest.platform-mesh.io/managed-by"

const managedByValue = "armament-sync"

// Syncer projects an external catalog onto Armament custom resources.
type Syncer struct {
	Client   client.Client
	Source   external.Client
	Interval time.Duration
}

// AddToManager registers the syncer's tick loop with the manager. Unlike a
// standard reconciler, this controller is driven entirely by a timer; there
// is no watch on Armament because the source of truth is external.
func (s *Syncer) AddToManager(mgr manager.Manager) error {
	if s.Interval <= 0 {
		return fmt.Errorf("sync interval must be > 0")
	}
	return mgr.Add(manager.RunnableFunc(s.run))
}

func (s *Syncer) run(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("armament-sync")
	logger.Info("starting armament sync loop", "interval", s.Interval)

	// Run an initial sync immediately so the catalog appears without
	// waiting a full interval after startup.
	if err := s.syncOnce(ctx); err != nil {
		logger.Error(err, "initial armament sync failed")
	}

	return wait.PollUntilContextCancel(ctx, s.Interval, false, func(ctx context.Context) (bool, error) {
		if err := s.syncOnce(ctx); err != nil {
			logger.Error(err, "armament sync iteration failed")
		}
		return false, nil
	})
}

func (s *Syncer) syncOnce(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("armament-sync")

	desired, err := s.Source.List(ctx)
	if err != nil {
		return fmt.Errorf("list from external source: %w", err)
	}

	existing := &wildwestv1alpha1.ArmamentList{}
	if err := s.Client.List(ctx, existing, client.MatchingLabels{managedByLabel: managedByValue}); err != nil {
		return fmt.Errorf("list managed armaments: %w", err)
	}

	existingByExternalID := make(map[string]*wildwestv1alpha1.Armament, len(existing.Items))
	for i := range existing.Items {
		a := &existing.Items[i]
		existingByExternalID[a.Spec.ExternalID] = a
	}

	desiredExternalIDs := make(map[string]struct{}, len(desired))
	for _, d := range desired {
		desiredExternalIDs[d.ExternalID] = struct{}{}
		if err := s.upsert(ctx, d); err != nil {
			logger.Error(err, "upsert armament", "externalID", d.ExternalID)
		}
	}

	for externalID, obj := range existingByExternalID {
		if _, kept := desiredExternalIDs[externalID]; kept {
			continue
		}
		if err := s.Client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			logger.Error(err, "delete stale armament", "name", obj.Name, "externalID", externalID)
		}
	}

	logger.V(1).Info("armament sync complete", "desired", len(desired), "existing", len(existing.Items))
	return nil
}

func (s *Syncer) upsert(ctx context.Context, src external.Armament) error {
	name := armamentName(src.ExternalID)
	armament := &wildwestv1alpha1.Armament{}
	err := s.Client.Get(ctx, types.NamespacedName{Name: name}, armament)
	switch {
	case apierrors.IsNotFound(err):
		armament = &wildwestv1alpha1.Armament{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: map[string]string{managedByLabel: managedByValue},
			},
			Spec: armamentSpecFromSource(src),
		}
		if err := s.Client.Create(ctx, armament); err != nil {
			return fmt.Errorf("create: %w", err)
		}
		return s.stampSyncTime(ctx, armament)
	case err != nil:
		return fmt.Errorf("get: %w", err)
	}

	desiredSpec := armamentSpecFromSource(src)
	if armament.Spec == desiredSpec && armament.Labels[managedByLabel] == managedByValue {
		return s.stampSyncTime(ctx, armament)
	}
	if armament.Labels == nil {
		armament.Labels = map[string]string{}
	}
	armament.Labels[managedByLabel] = managedByValue
	armament.Spec = desiredSpec
	if err := s.Client.Update(ctx, armament); err != nil {
		return fmt.Errorf("update: %w", err)
	}
	return s.stampSyncTime(ctx, armament)
}

func (s *Syncer) stampSyncTime(ctx context.Context, armament *wildwestv1alpha1.Armament) error {
	now := metav1.NewTime(time.Now())
	armament.Status.LastSyncedAt = &now
	if err := s.Client.Status().Update(ctx, armament); err != nil {
		return fmt.Errorf("status update: %w", err)
	}
	return nil
}

func armamentSpecFromSource(src external.Armament) wildwestv1alpha1.ArmamentSpec {
	return wildwestv1alpha1.ArmamentSpec{
		ExternalID:  src.ExternalID,
		DisplayName: src.DisplayName,
		Kind:        src.Kind,
		Damage:      src.Damage,
		Range:       src.Range,
	}
}

// armamentName turns an opaque external identifier into a DNS-safe object
// name. The external system may use punctuation that kubernetes object
// names disallow, so we sanitize in one place.
func armamentName(externalID string) string {
	out := make([]byte, 0, len(externalID))
	for i := 0; i < len(externalID); i++ {
		c := externalID[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-':
			out = append(out, c)
		case c >= 'A' && c <= 'Z':
			out = append(out, c+('a'-'A'))
		default:
			out = append(out, '-')
		}
	}
	if len(out) == 0 {
		return "armament"
	}
	return string(out)
}
