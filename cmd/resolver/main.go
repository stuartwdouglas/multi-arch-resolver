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
	"fmt"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resolution/v1beta1"
	"github.com/tektoncd/pipeline/pkg/resolution/common"
	"github.com/tektoncd/pipeline/pkg/resolution/resolver/bundle"
	"github.com/tektoncd/pipeline/pkg/resolution/resolver/framework"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	filteredinformerfactory "knative.dev/pkg/client/injection/kube/informers/factory/filtered"
	"knative.dev/pkg/injection/sharedmain"
	"strings"
)

const ArchParam = "arch"
const ConfigMapName = "bundleresolver-config"

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

// GetConfigName returns the name of the bundle resolver's configmap.
func (r *resolver) GetConfigName(context.Context) string {
	return ConfigMapName
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

	conf := framework.GetResolverConfigFromContext(ctx)

	paramsMap := make(map[string]pipelinev1beta1.ParamValue)
	for _, p := range params {
		paramsMap[p.Name] = p.Value
	}

	saVal, ok := paramsMap[bundle.ParamServiceAccount]
	sa := ""
	if !ok || saVal.StringVal == "" {
		if saString, ok := conf[bundle.ConfigServiceAccount]; ok {
			sa = saString
		} else {
			return nil, fmt.Errorf("default Service Account was not set during installation of the multi-arch resolver")
		}
	} else {
		sa = saVal.StringVal
	}
	bundleParams = append(bundleParams, pipelinev1beta1.Param{Name: bundle.ParamServiceAccount, Value: pipelinev1beta1.ParamValue{StringVal: sa}})

	bundleResult, err := r.bundleResolver.Resolve(ctx, bundleParams)
	if err != nil {
		return bundleResult, err
	}

	decodingScheme := runtime.NewScheme()
	utilruntime.Must(pipelinev1beta1.AddToScheme(decodingScheme))
	decoderCodecFactory := serializer.NewCodecFactory(decodingScheme)
	decoder := decoderCodecFactory.UniversalDecoder(pipelinev1beta1.SchemeGroupVersion)
	task := pipelinev1beta1.Task{}
	err = runtime.DecodeInto(decoder, bundleResult.Data(), &task)
	if err != nil {
		return nil, err
	}

	convertToSsh(&task)

	println("Data: " + task.Spec.Steps[0].Name)
	codec := serializer.NewCodecFactory(decodingScheme).LegacyCodec(pipelinev1beta1.SchemeGroupVersion)
	output, err := runtime.Encode(codec, &task)
	if err != nil {
		return nil, err
	}
	return &resolvedResource{
		data:        output,
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

//script
//set 1 sets up the ssh server

func convertToSsh(task *pipelinev1beta1.Task) string {

	podmanArgs := ""

	ret := `#!/bin/bash
set -o verbose
mkdir ~/.ssh
cp /ssh/id_rsa ~/.ssh
chmod 0400 ~/.ssh/id_rsa
export SSH_HOST=ec2-user@ec2-44-211-78-24.compute-1.amazonaws.com
export SSH_ARGS="-o StrictHostKeyChecking=no"
export BUILD_ID=tmpbuildid
export BUILD_DIR=/tmp/fakebuild
rm -rf $BUILD_DIR
mkdir -p scripts
ssh -o StrictHostKeyChecking=no $SSH_ARGS $SSH_HOST  mkdir -p $BUILD_DIR/workspaces $BUILD_DIR/scripts`
	//before the build we sync the contents of the workspace to the remote host
	for _, workspace := range task.Spec.Workspaces {
		ret += "\nrsync -ra $(workspaces." + workspace.Name + ".path) $SSH_HOST:$BUILD_DIR/workspaces/" + workspace.Name
		podmanArgs += " -v $BUILD_DIR/workspaces/" + workspace.Name + ":/$(workspaces." + workspace.Name + ".path):Z "
	}
	for _, step := range task.Spec.Steps {
		script := "scripts/script-" + step.Name + ".sh"

		ret += "\ncat >" + script + " <<'REMOTESSHEOF'\n"
		if !strings.HasPrefix(step.Script, "#!") {
			ret += "#!/bin/sh\n"
		}
		ret += step.Script
		ret += "\nREMOTESSHEOF"
		ret += "\nchmod +x " + script
	}
	ret += "\nrsync -ra scripts $SSH_HOST:$BUILD_DIR"
	for _, step := range task.Spec.Steps {
		script := "/script/script-" + step.Name + ".sh"
		ret += "\nssh $SSH_ARGS $SSH_HOST podman  run --rm " + podmanArgs + " -v $BUILD_DIR/scripts:/script:Z --user=0  " + replaceImage(step.Image) + "  " + script
	}

	//sync the contents of the workspaces back so subsequent tasks can use them
	for _, workspace := range task.Spec.Workspaces {
		ret += "\nrsync -ra $SSH_HOST:$BUILD_DIR/workspaces/ $(workspaces." + workspace.Name + ".path) "
	}

	task.Spec.Volumes = append(task.Spec.Volumes, v1.Volume{
		Name: "ssh",
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: "ssh",
			},
		},
	})
	task.Spec.Steps = []pipelinev1beta1.Step{
		{
			Name:            "run-remote-build",
			Image:           "quay.io/sdouglas/registry:multiarch",
			ImagePullPolicy: v1.PullAlways,
			Script:          ret,
			SecurityContext: &v1.SecurityContext{RunAsUser: new(int64)},
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      "ssh",
					ReadOnly:  true,
					MountPath: "/ssh",
				},
			},
		},
	}
	return ret
}

func replaceImage(image string) string {
	if image == "quay.io/redhat-appstudio/buildah:v1.28" {
		return "quay.io/buildah/stable:v1.28"
	}
	return image
}
