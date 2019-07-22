#!/bin/bash

kubectl delete -f deploy/crds/example_v1alpha1_nginx_cr.yaml
kubectl delete -f deploy/operator.yaml
kubectl delete -f deploy/crds/example_v1alpha1_nginx_crd.yaml
echo "Deleted crd, operator and cr"

/usr/local/bin/operator-sdk build ameydev/nginx-operator:latest
docker push ameydev/nginx-operator:latest

kubectl create -f deploy/crds/example_v1alpha1_nginx_crd.yaml
kubectl create -f deploy/operator.yaml
kubectl create -f deploy/crds/example_v1alpha1_nginx_cr.yaml


