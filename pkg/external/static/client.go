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

// Package static is a stand-in external.Client backed by a hardcoded list,
// used for development and tests until a real backend is wired up.
package static

import (
	"context"

	"github.com/platform-mesh/provider-quickstart/pkg/external"
)

// Client returns the same hardcoded armaments on every call.
type Client struct{}

// New returns a static external.Client.
func New() *Client { return &Client{} }

// List returns the hardcoded armament catalog.
func (c *Client) List(_ context.Context) ([]external.Armament, error) {
	return []external.Armament{
		{ExternalID: "colt-saa", DisplayName: "Colt Single Action Army", Kind: "revolver", Damage: 50, Range: 50},
		{ExternalID: "winchester-1873", DisplayName: "Winchester Model 1873", Kind: "rifle", Damage: 80, Range: 400},
		{ExternalID: "lasso", DisplayName: "Lasso", Kind: "rope", Damage: 5, Range: 10},
		{ExternalID: "bowie-knife", DisplayName: "Bowie Knife", Kind: "blade", Damage: 30, Range: 2},
	}, nil
}
