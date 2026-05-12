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

package crds

import (
	"embed"
)

// ProviderFS holds CRDs that must be installed in the provider workspace
// itself (not exposed via APIExport with crd:{} storage). Armaments are
// stored in the provider workspace and replicated to consumers as read-only
// via a CachedResource, so the provider workspace needs the real CRD.
//
//go:embed wildwest.platform-mesh.io_armaments.yaml
var ProviderFS embed.FS
