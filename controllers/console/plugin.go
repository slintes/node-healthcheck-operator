/*
Copyright 2022.

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

package console

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	v1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/api/console/v1alpha1"

	"github.com/medik8s/node-healthcheck-operator/controllers/utils"
)

const (
	// PluginName is the name of the plugin and used at several places
	PluginName = "node-remediation-console-plugin"
	// ServiceName is the name of the console plugin Service and must match the name of the Service in /bundle/manifests!
	ServiceName = "node-healthcheck-node-remediation-console-plugin"
	// ServicePort is the port of the console plugin Service and must match the port of the Service in /bundle/manifests!
	ServicePort = 9443
)

// +kubebuilder:rbac:groups=console.openshift.io,resources=consoleplugins,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

// CreateOrUpdatePlugin creates or updates the resources needed for the remediation console plugin.
// HEADS UP: consider cleanup of old resources in case of name changes or removals!
//
// TODO image reference to console plugin in CSV?
func CreateOrUpdatePlugin(ctx context.Context, cl client.Client, config *rest.Config, namespace string, log logr.Logger) error {

	// check if we are on OCP
	if isOpenshift, err := utils.IsOnOpenshift(config); err != nil {
		return errors.Wrap(err, "failed to check if we are on Openshift")
	} else if !isOpenshift {
		log.Info("we are not on Openshift, skipping console plugin activation, but scale down console deployment")
		list := &v1.DeploymentList{}
		selector := labels.NewSelector()
		req, _ := labels.NewRequirement("app.kubernetes.io/component", selection.Equals, []string{"node-remediation-console-plugin"})
		selector = selector.Add(*req)
		if err := cl.List(ctx, list, &client.ListOptions{LabelSelector: selector, Namespace: namespace}); err != nil || len(list.Items) != 1 {
			return errors.Wrap(err, "failed to find console deployment for scaling down")
		}
		console := list.Items[0]
		consoleOrig := console.DeepCopy()
		replicas := int32(0)
		console.Spec.Replicas = &replicas
		mergeFrom := client.MergeFrom(consoleOrig)
		if err = cl.Patch(ctx, &console, mergeFrom, &client.PatchOptions{}); err != nil {
			return errors.Wrap(err, "failed to patch console deployment for scaling down")
		}
		return nil
	}

	// Create ConsolePlugin resource
	// Deployment and Service are deployed by OLM
	if err := createOrUpdateConsolePlugin(ctx, namespace, cl); err != nil {
		return err
	}

	log.Info("successfully created / updated console plugin resources")
	return nil
}

func createOrUpdateConsolePlugin(ctx context.Context, namespace string, cl client.Client) error {
	cp := getConsolePlugin(namespace)
	oldCP := &v1alpha1.ConsolePlugin{}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(cp), oldCP); apierrors.IsNotFound(err) {
		if err := cl.Create(ctx, cp); err != nil {
			return errors.Wrap(err, "could not create console plugin")
		}
	} else if err != nil {
		return errors.Wrap(err, "could not check for existing console plugin")
	} else {
		oldCP.OwnerReferences = cp.OwnerReferences
		oldCP.Spec = cp.Spec
		if err := cl.Update(ctx, oldCP); err != nil {
			return errors.Wrap(err, "could not update console plugin")
		}
	}
	return nil
}

func getConsolePlugin(namespace string) *v1alpha1.ConsolePlugin {
	return &v1alpha1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: PluginName,
			// TODO set owner ref for deletion when operator is uninstalled
			// but which resource to use, needs to be cluster scoped
		},
		Spec: v1alpha1.ConsolePluginSpec{
			DisplayName: "Node Remediation",
			Service: v1alpha1.ConsolePluginService{
				Name:      ServiceName,
				Namespace: namespace,
				Port:      ServicePort,
				BasePath:  "/",
			},
		},
	}
}
