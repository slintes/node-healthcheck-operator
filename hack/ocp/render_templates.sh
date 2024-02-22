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
# !!! keep aligned with the channels in the Dockerfile.in !!!!
#export CHANNELS="4.14-eus,stable"
export CHANNELS="stable"

# skip range lower boundary, set to last eus version
# 0.6.x = 4.14-eus
export SKIP_RANGE_LOWER="0.6.0"
# For the replaces field in the CSV, set to the last released version
# Prevents removal of that version from index
export PREVIOUS_VERSION="0.6.1"

##################
### Fixes vars ###
##################

#UPSTREAM_DIR="upstream"
export MANIFESTS_DIR="bundle/manifests"
export METADATA_DIR="bundle/metadata"
export TESTS_DIR="bundle/tests"
export BUILD_REGISTRY="registry-proxy.engineering.redhat.com/rh-osbs/red-hat-workload-availability"
export OPERATOR_NAME="node-healthcheck-operator"
export CONSOLE_OPERATOR_NAME="node-remediation-console"
export MUST_GATHER_NAME="node-healthcheck-must-gather-rhel8"
export CSV=${MANIFESTS_DIR}/${OPERATOR_NAME}.clusterserviceversion.yaml
export ANNOTATIONS=${METADATA_DIR}/annotations.yaml

# cd into project dir
cd ../..
pwd

# Make bundle
rm -rf bundle
make bundle-ocp

exit 0

# Update image URLs
# Note: "mage: ..." matches both image and containerImage
sed -r -i "s|mage: quay.io/medik8s/${OPERATOR_NAME}.*|mage: ${BUILD_REGISTRY}-${OPERATOR_NAME}:v${CI_VERSION}|g;" "${CSV}"
sed -r -i "s|mage: quay.io/medik8s/${CONSOLE_OPERATOR_NAME}.*|mage: ${BUILD_REGISTRY}-${CONSOLE_OPERATOR_NAME}:v${CI_VERSION}|g;" "${CSV}"
sed -r -i "s|mage: .*/kube-rbac-proxy.*|mage: registry.redhat.io/openshift4/ose-kube-rbac-proxy:v${RBAC_PROXY_OCP_VERSION}|g;" "${CSV}"

# Add env var with must gather image to the NHC container, so its pullspec gets added to the relatedImages section by OSBS
# https://osbs.readthedocs.io/en/osbs_ocp3/users.html?#pinning-pullspecs-for-related-images
sed -r -i "/                - name: DEPLOYMENT_NAMESPACE/ i\                - name: RELATED_IMAGE_MUST_GATHER" ${CSV}
sed -r -i "/                - name: DEPLOYMENT_NAMESPACE/ i\                  value: ${BUILD_REGISTRY}-${MUST_GATHER_NAME}:v${CI_VERSION}" ${CSV}

# Update build date
sed -r -i "s|createdAt: .*|createdAt: `date '+%Y-%m-%d %T'`|;" "${CSV}"

# Update version
sed -r -i "s|name: ${OPERATOR_NAME}.v.*|name: ${OPERATOR_NAME}.v${CI_VERSION}|;" "${CSV}"
sed -r -i "s|version: 0.0.1|version: ${CI_VERSION}|;" "${CSV}"
sed -r -i "s|skipRange: .*|skipRange: '>=${SKIP_RANGE_LOWER} <${CI_VERSION}'|;" "${CSV}"
# add replaces field
sed -r -i "/  version: ${CI_VERSION}/ a\  replaces: ${OPERATOR_NAME}.v${PREVIOUS_VERSION}" ${CSV}

# Update Medik8s to Dragonfly
sed -r -i "s|email: medik8s@googlegroups.com|email: team-dragonfly@redhat.com|;" "${CSV}"
sed -r -i "s|name: Medik8s Team|name: Dragonfly Team|;" "${CSV}"

# Update Medik8s to Red Hat
sed -r -i "s|support: Medik8s|support: Red Hat|;" "${CSV}"
sed -r -i "s|  name: Medik8s|  name: Red Hat|;" "${CSV}"
sed -r -i -z "s|url: https://github.com/medik8s|url: https://www.redhat.com|2;" "${CSV}"

# Add OCP annotations
sed -r -i "/    olm.skipRange:.*/a \    operators.openshift.io/valid-subscription: '[\"OpenShift Kubernetes Engine\", \"OpenShift Container Platform\", \"OpenShift Platform Plus\"]'" "${CSV}"
sed -r -i "/    olm.skipRange:.*/a \    operators.openshift.io/infrastructure-features: '[\"disconnected\"]'" "${CSV}"
# Even more now / others now, above are deprecated, see https://docs.engineering.redhat.com/display/CFC/Best_Practices#Best_Practices-(New)RequiredInfrastructureAnnotations
sed -r -i "/    olm.skipRange:.*/a \    features.operators.openshift.io/disconnected: 'true'" "${CSV}"
sed -r -i "/    olm.skipRange:.*/a \    features.operators.openshift.io/fips-compliant: 'false'" "${CSV}"
sed -r -i "/    olm.skipRange:.*/a \    features.operators.openshift.io/proxy-aware: 'false'" "${CSV}"
sed -r -i "/    olm.skipRange:.*/a \    features.operators.openshift.io/tls-profiles: 'false'" "${CSV}"
sed -r -i "/    olm.skipRange:.*/a \    features.operators.openshift.io/token-auth-aws: 'false'" "${CSV}"
sed -r -i "/    olm.skipRange:.*/a \    features.operators.openshift.io/token-auth-azure: 'false'" "${CSV}"
sed -r -i "/    olm.skipRange:.*/a \    features.operators.openshift.io/token-auth-gcp: 'false'" "${CSV}"

# Update URL link to OCP docs
sed -r -i "s| url: https://medik8s.io| url: https://access.redhat.com/documentation/en-us/workload_availability_for_red_hat_openshift/${DOCS_RHWA_VERSION}/html/remediation_fencing_and_maintenance/node-health-check-operator|;" "${CSV}"

# Set downstream (red) icon
redIcon=$(base64 --wrap=0 ./config/assets/nhc_red.png)
sed -r -i "s|  - base64data: .*|  - base64data: $redIcon|;" "${CSV}"

# Update Channels for annotations.yaml file - EUS version
sed -r -i "s|channels.v1:.*|channels.v1: ${CHANNELS}|;" "${ANNOTATIONS}"
