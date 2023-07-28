#!/bin/bash

DIR=`dirname $0`

kubectl delete resolutionrequests.resolution.tekton.dev --all

kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.47.3/release.yaml
eval $(minikube -p minikube docker-env)
docker build . -t quay.io/${QUAY_USERNAME}/multi-arch-resolver:dev
kubectl apply -f $DIR/resolver-deployment.yaml
kubectl rollout restart deployment -n tekton-pipelines-resolvers multi-arch-resolver

kubectl create -f $DIR/test-resolver-template.yaml
