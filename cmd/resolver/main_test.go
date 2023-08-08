package main

import (
	"bytes"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"os"
	"path/filepath"
	"testing"
)

func TestExampleTask(t *testing.T) {
	task := pipelinev1beta1.Task{}
	streamFileYamlToTektonObj("/home/stuart/workspace/build-definitions/task/buildah/0.1/buildah.yaml", &task)

	decodingScheme := runtime.NewScheme()
	utilruntime.Must(pipelinev1beta1.AddToScheme(decodingScheme))
	convertToSsh(&task)
	y := printers.YAMLPrinter{}
	b := bytes.Buffer{}
	_ = y.PrintObj(&task, &b)
	err := os.WriteFile("/home/stuart/workspace/build-definitions/task/buildah-remote/0.1/buildah-remote.yaml", b.Bytes(), 0660)
	if err != nil {
		panic(err)
	}
}

func decodeBytesToTektonObjbytes(bytes []byte, obj runtime.Object) runtime.Object {
	decodingScheme := runtime.NewScheme()
	utilruntime.Must(pipelinev1beta1.AddToScheme(decodingScheme))
	decoderCodecFactory := serializer.NewCodecFactory(decodingScheme)
	decoder := decoderCodecFactory.UniversalDecoder(pipelinev1beta1.SchemeGroupVersion)
	err := runtime.DecodeInto(decoder, bytes, obj)
	if err != nil {
		panic(err)
	}
	return obj
}

func streamFileYamlToTektonObj(path string, obj runtime.Object) runtime.Object {
	bytes, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		panic(err)
	}
	return decodeBytesToTektonObjbytes(bytes, obj)
}
