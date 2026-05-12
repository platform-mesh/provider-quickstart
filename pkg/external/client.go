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

// Package external defines the external-source contract that the
// armament-sync controller consumes. The provider workspace is treated as a
// projection of whatever the external system reports; this package is the
// seam where a real backend can be substituted.
package external

import "context"

// Armament is the external-source representation of a catalog item. It is
// intentionally decoupled from the kubernetes API type so that backend
// changes do not ripple through to the CRD.
type Armament struct {
	ExternalID  string
	DisplayName string
	Kind        string
	Damage      int32
	Range       int32
}

// Client lists the full set of armaments currently available from the
// external system. Implementations must return the complete authoritative
// set on each call; the sync controller diffs that set against what is
// stored in the provider workspace.
type Client interface {
	List(ctx context.Context) ([]Armament, error)
}
