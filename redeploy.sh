#!/bin/bash
set -x

make build-template-validator-container push-template-validator-container container-build container-push
oc delete clusterrolebinding ssp-operator-rolebinding
oc delete clusterrole ssp-operator-role
oc delete deployment.apps/ssp-operator  -n kubevirt
make deploy
