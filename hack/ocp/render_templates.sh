#!/bin/bash

set -eux

####################################################
### UPDATE VARS IN THIS SECTION FOR EACH RELEASE ###
####################################################

# For the docs URL
export DOCS_RHWA_VERSION="24.1"
# For the rbac proxy image; only works after release, so it's one release behind most times...
export RBAC_PROXY_OCP_VERSION="4.14"

# Bundle channels
export CHANNELS="4.14-eus,stable"

# skip range lower boundary, set to last eus version
export SKIP_RANGE_LOWER="0.6.0"
# For the replaces field in the CSV, set to the last released version
# Prevents removal of that version from index
export PREVIOUS_VERSION="0.6.1"

# prevent conflics with Makefile vars here!
export BUILD_REGISTRY="registry-proxy.engineering.redhat.com/rh-osbs/red-hat-workload-availability"
export OPERATOR_IMAGE_NAME="node-healthcheck-operator"
export CONSOLE_PLUGIN_IMAGE_NAME="node-remediation-console"
export MUST_GATHER_IMAGE_NAME="node-healthcheck-must-gather-rhel8"

# cd into project dir
cd ../..
pwd

# Override Makefile variables
export VERSION=${CI_VERSION}
export IMG=${BUILD_REGISTRY}-${OPERATOR_IMAGE_NAME}:v${CI_VERSION}
export CONSOLE_PLUGIN_IMAGE=${BUILD_REGISTRY}-${CONSOLE_PLUGIN_IMAGE_NAME}:v${CI_VERSION}
export RBAC_PROXY_IMAGE="registry.redhat.io/openshift4/ose-kube-rbac-proxy:v${RBAC_PROXY_OCP_VERSION}"
export MUST_GATHER_IMAGE=${BUILD_REGISTRY}-${MUST_GATHER_IMAGE_NAME}:v${CI_VERSION}

# Make bundle
rm -rf bundle
make bundle-ocp
