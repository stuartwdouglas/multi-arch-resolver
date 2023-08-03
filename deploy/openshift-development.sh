#!/bin/bash

DIR=`dirname $0`

kubectl delete resolutionrequests.resolution.tekton.dev --all

docker build . -t quay.io/${QUAY_USERNAME}/multi-arch-resolver:dev
docker push quay.io/${QUAY_USERNAME}/multi-arch-resolver:dev
kubectl apply -f $DIR/resolver-openshift-deployment.yaml
kubectl rollout restart deployment -n openshift-pipelines multi-arch-resolver

sleep 10
