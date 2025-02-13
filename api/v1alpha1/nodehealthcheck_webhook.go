/*
Copyright 2021.

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

package v1alpha1

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	WebhookCertDir  = "/apiserver.local.config/certificates"
	WebhookCertName = "apiserver.crt"
	WebhookKeyName  = "apiserver.key"

	OngoingRemediationError = "prohibited due to running remediation"
	minHealthyError         = "MinHealthy must not be negative"
	invalidSelectorError    = "Invalid selector"
)

// log is for logging in this package.
var nodehealthchecklog = logf.Log.WithName("nodehealthcheck-resource")

func (r *NodeHealthCheck) SetupWebhookWithManager(mgr ctrl.Manager) error {

	// check if OLM injected certs
	certs := []string{filepath.Join(WebhookCertDir, WebhookCertName), filepath.Join(WebhookCertDir, WebhookKeyName)}
	certsInjected := true
	for _, fname := range certs {
		if _, err := os.Stat(fname); err != nil {
			certsInjected = false
			break
		}
	}
	if certsInjected {
		server := mgr.GetWebhookServer()
		server.CertDir = WebhookCertDir
		server.CertName = WebhookCertName
		server.KeyName = WebhookKeyName
	} else {
		nodehealthchecklog.Info("OLM injected certs for webhooks not found")
	}

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-remediation-medik8s-io-v1alpha1-nodehealthcheck,mutating=false,failurePolicy=fail,sideEffects=None,groups=remediation.medik8s.io,resources=nodehealthchecks,verbs=update;delete,versions=v1alpha1,name=vnodehealthcheck.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NodeHealthCheck{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NodeHealthCheck) ValidateCreate() error {
	nodehealthchecklog.Info("validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NodeHealthCheck) ValidateUpdate(old runtime.Object) error {
	nodehealthchecklog.Info("validate update", "name", r.Name)

	// do the normal validation
	if err := r.validate(); err != nil {
		return err
	}

	// during ongoing remediations, some updates are forbidden
	if r.isRemediating() {
		if updated, field := r.isRestrictedFieldUpdated(old.(*NodeHealthCheck)); updated {
			return fmt.Errorf("%s update %s", field, OngoingRemediationError)
		}
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NodeHealthCheck) ValidateDelete() error {
	nodehealthchecklog.Info("validate delete", "name", r.Name)
	if r.isRemediating() {
		return fmt.Errorf("deletion %s", OngoingRemediationError)
	}
	return nil
}

func (r *NodeHealthCheck) validate() error {
	aggregated := errors.NewAggregate([]error{r.validateMinHealthy(), r.validateSelector()})

	// everything else should have been covered by API server validation
	// as defined by kubebuilder validation markers on the NHC struct.

	return aggregated
}

func (r *NodeHealthCheck) validateMinHealthy() error {
	// Using Minimum kubebuilder marker for IntOrStr does not work (yet)
	if r.Spec.MinHealthy == nil {
		return fmt.Errorf("MinHealthy must not be empty")
	}
	if r.Spec.MinHealthy.Type == intstr.Int && r.Spec.MinHealthy.IntVal < 0 {
		return fmt.Errorf("%s: %v", minHealthyError, r.Spec.MinHealthy)
	}
	return nil
}

func (r *NodeHealthCheck) validateSelector() error {
	if _, err := metav1.LabelSelectorAsSelector(&r.Spec.Selector); err != nil {
		return fmt.Errorf("%s: %v", invalidSelectorError, err.Error())
	}
	return nil
}

func (r *NodeHealthCheck) isRestrictedFieldUpdated(old *NodeHealthCheck) (bool, string) {
	// modifying these fields can cause dangling remediations
	if !reflect.DeepEqual(r.Spec.Selector, old.Spec.Selector) {
		return true, "selector"
	}
	if !reflect.DeepEqual(r.Spec.RemediationTemplate, old.Spec.RemediationTemplate) {
		return true, "remediation template"
	}
	return false, ""
}

func (r *NodeHealthCheck) isRemediating() bool {
	return len(r.Status.InFlightRemediations) > 0
}
