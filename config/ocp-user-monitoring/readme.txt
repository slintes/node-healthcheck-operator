Enable user workload monitoring on OCP by applying these manifests.
Use with care, don't override existing configs :)

Create prometheus user workload token secret:

kubectl get secret prometheus-user-workload-token-<SOME_RANDOM_CHARS> --namespace=openshift-user-workload-monitoring -o yaml | \
    sed '/namespace: .*==/d;/ca.crt:/d;/serviceCa.crt/d;/creationTimestamp:/d;/resourceVersion:/d;/uid:/d;/annotations/d;/kubernetes.io/d;' | \
    sed 's/namespace: .*/namespace: openshift-operators/' \
    sed 's/name: .*/name: prometheus-user-workload-token/' | \
    sed 's/type: .*/type: Opaque/' | \
    > prom-token.yaml

kubectl apply -f prom-token.yaml
