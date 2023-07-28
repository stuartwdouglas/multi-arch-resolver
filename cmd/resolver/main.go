/*
Copyright 2022 The Tekton Authors
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
	"errors"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resolution/v1beta1"
	"github.com/tektoncd/pipeline/pkg/resolution/common"
	"github.com/tektoncd/pipeline/pkg/resolution/resolver/bundle"
	"github.com/tektoncd/pipeline/pkg/resolution/resolver/framework"
	filteredinformerfactory "knative.dev/pkg/client/injection/kube/informers/factory/filtered"
	"knative.dev/pkg/injection/sharedmain"
)

const ArchParam = "arch"

func main() {
	ctx := filteredinformerfactory.WithSelectors(context.Background(), v1beta1.ManagedByLabelKey)
	sharedmain.MainWithContext(ctx, "controller",
		framework.NewController(ctx, &resolver{}),

	)
}

type resolver struct {
	bundleResolver bundle.Resolver
}

// Initialize sets up any dependencies needed by the resolver. None atm.
func (r *resolver) Initialize(c context.Context) error {
	r.bundleResolver.Initialize(c)
	return nil
}

// GetName returns a string name to refer to this resolver by.
func (r *resolver) GetName(context.Context) string {
	return "Demo"
}

// GetSelector returns a map of labels to match requests to this resolver.
func (r *resolver) GetSelector(context.Context) map[string]string {
	return map[string]string{
		common.LabelKeyResolverType: "multi-arch-bundle",
	}
}

// ValidateParams ensures parameters from a request are as expected.
func (r *resolver) ValidateParams(ctx context.Context, params []pipelinev1beta1.Param) error {
	arch, bundleParams := extractParams(params)
	if arch == "" {
		return errors.New("arch param is required")
	}
	return r.bundleResolver.ValidateParams(ctx, bundleParams)
}

// Resolve uses the given params to resolve the requested file or resource.
func (r *resolver) Resolve(ctx context.Context, params []pipelinev1beta1.Param) (framework.ResolvedResource, error) {
	arch, bundleParams := extractParams(params)
	println(arch)
	bundleResult, err := r.bundleResolver.Resolve(ctx, bundleParams)
	if err != nil {
		return bundleResult, err
	}
	println("Data: " + string(bundleResult.Data()))

	return &resolvedResource{
		data:        bundleResult.Data(),
		annotations: bundleResult.Annotations(),
		refSource:   bundleResult.RefSource(),
	}, nil
}

func extractParams(params []pipelinev1beta1.Param) (string, []pipelinev1beta1.Param) {
	bundleParams := []pipelinev1beta1.Param{}
	arch := ""
	for _, p := range params {
		if p.Name == ArchParam {
			arch = p.Value.StringVal
		} else {
			bundleParams = append(bundleParams, p)
		}
	}
	return arch, bundleParams
}

// resolvedResource wraps the data we want to return to Pipelines
type resolvedResource struct {
	data        []byte
	annotations map[string]string
	refSource   *pipelinev1beta1.RefSource
}

// Data returns the bytes of our hard-coded Pipeline
func (r *resolvedResource) Data() []byte {
	return r.data
}

// Annotations returns any metadata needed alongside the data. None atm.
func (r *resolvedResource) Annotations() map[string]string {
	return r.annotations
}

// RefSource is the source reference of the remote data that records where the remote
// file came from including the url, digest and the entrypoint. None atm.
func (r *resolvedResource) RefSource() *pipelinev1beta1.RefSource {
	return r.refSource
}
