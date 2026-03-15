/*
Copyright 2026.

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

package controller

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	securityv1 "github.com/aldarisbm/wiz-ccr/api/v1"
	"github.com/aldarisbm/wiz-ccr/internal/wiz"
)

const finalizer = "wiz.joseberr.io/finalizer"

// WizCloudConfigurationRuleReconciler reconciles a WizCloudConfigurationRule object
type WizCloudConfigurationRuleReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	WizClient wiz.Client
}

// +kubebuilder:rbac:groups=security.joseberr.io,resources=wizcloudconfigurationrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.joseberr.io,resources=wizcloudconfigurationrules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=security.joseberr.io,resources=wizcloudconfigurationrules/finalizers,verbs=update

func (r *WizCloudConfigurationRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var rule securityv1.WizCloudConfigurationRule
	if err := r.Get(ctx, req.NamespacedName, &rule); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// --- Deletion ---
	if !rule.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&rule, finalizer) {
			if rule.Status.WizRuleID != "" {
				if err := r.WizClient.DeleteRule(ctx, rule.Status.WizRuleID); err != nil {
					log.Error(err, "Failed to delete rule from Wiz", "wizRuleID", rule.Status.WizRuleID)
					return ctrl.Result{}, err
				}
				log.Info("Deleted rule from Wiz", "wizRuleID", rule.Status.WizRuleID)
			}
			controllerutil.RemoveFinalizer(&rule, finalizer)
			if err := r.Update(ctx, &rule); err != nil {
				return ctrl.Result{}, fmt.Errorf("removing finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}

	// --- Ensure finalizer is present ---
	if !controllerutil.ContainsFinalizer(&rule, finalizer) {
		controllerutil.AddFinalizer(&rule, finalizer)
		if err := r.Update(ctx, &rule); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		// Re-fetch after update so status patch uses the latest resource version.
		if err := r.Get(ctx, req.NamespacedName, &rule); err != nil {
			return ctrl.Result{}, err
		}
	}

	// --- Create or update the Wiz rule ---
	wizRule := specToRule(rule.Spec)

	if rule.Status.WizRuleID == "" {
		id, err := r.WizClient.CreateRule(ctx, wizRule)
		if err != nil {
			log.Error(err, "Failed to create rule in Wiz")
			return ctrl.Result{}, r.setDegraded(ctx, &rule, fmt.Sprintf("create failed: %v", err))
		}
		log.Info("Created rule in Wiz", "wizRuleID", id)
		rule.Status.WizRuleID = id
	} else {
		if err := r.WizClient.UpdateRule(ctx, rule.Status.WizRuleID, wizRule); err != nil {
			log.Error(err, "Failed to update rule in Wiz", "wizRuleID", rule.Status.WizRuleID)
			return ctrl.Result{}, r.setDegraded(ctx, &rule, fmt.Sprintf("update failed: %v", err))
		}
		log.Info("Updated rule in Wiz", "wizRuleID", rule.Status.WizRuleID)
	}

	return ctrl.Result{}, r.setAvailable(ctx, &rule)
}

// setAvailable writes an Available=True condition and persists status.
func (r *WizCloudConfigurationRuleReconciler) setAvailable(ctx context.Context, rule *securityv1.WizCloudConfigurationRule) error {
	rule.Status.Conditions = setCondition(rule.Status.Conditions, metav1.Condition{
		Type:               "Available",
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciled",
		Message:            "Rule is in sync with Wiz",
		ObservedGeneration: rule.Generation,
	})
	if err := r.Status().Update(ctx, rule); err != nil {
		return fmt.Errorf("updating status to Available: %w", err)
	}
	return nil
}

// setDegraded writes a Degraded=True condition, persists status, and returns the original error.
func (r *WizCloudConfigurationRuleReconciler) setDegraded(ctx context.Context, rule *securityv1.WizCloudConfigurationRule, msg string) error {
	rule.Status.Conditions = setCondition(rule.Status.Conditions, metav1.Condition{
		Type:               "Degraded",
		Status:             metav1.ConditionTrue,
		Reason:             "WizAPIError",
		Message:            msg,
		ObservedGeneration: rule.Generation,
	})
	if err := r.Status().Update(ctx, rule); err != nil {
		return fmt.Errorf("updating status to Degraded: %w", err)
	}
	return fmt.Errorf("%s", msg)
}

// setCondition upserts a condition into the list by type.
func setCondition(conditions []metav1.Condition, next metav1.Condition) []metav1.Condition {
	next.LastTransitionTime = metav1.Now()
	for i, c := range conditions {
		if c.Type == next.Type {
			conditions[i] = next
			return conditions
		}
	}
	return append(conditions, next)
}

// specToRule converts a CRD spec into the wiz.Rule shape the client expects.
func specToRule(spec securityv1.WizCloudConfigurationRuleSpec) wiz.Rule {
	var name, description, projectScope, targetNativeType, code, remediationSteps string
	if spec.RuleName != nil {
		name = *spec.RuleName
	}
	if spec.Description != nil {
		description = *spec.Description
	}
	if spec.ProjectScope != nil {
		projectScope = *spec.ProjectScope
	}
	if spec.TargetNativeType != nil {
		targetNativeType = *spec.TargetNativeType
	}
	if spec.Code != nil {
		code = *spec.Code
	}
	if spec.RemediationSteps != nil {
		remediationSteps = *spec.RemediationSteps
	}

	opTypes := make([]string, len(spec.OperationTypes))
	for i, op := range spec.OperationTypes {
		opTypes[i] = string(op)
	}

	matcherTypes := make([]wiz.MatcherType, len(spec.Matchers))
	for i, m := range spec.Matchers {
		matcherTypes[i] = wiz.MatcherType(m)
	}

	return wiz.Rule{
		Name:                name,
		Description:         description,
		Severity:            string(spec.FindingSeverity),
		ProjectScope:        projectScope,
		FrameworkCategories: spec.FrameworkCategories,
		Tags:                spec.Tags,
		TargetNativeType:    targetNativeType,
		MatcherTypes:        matcherTypes,
		OperationTypes:      opTypes,
		Code:                code,
		RemediationSteps:    remediationSteps,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *WizCloudConfigurationRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&securityv1.WizCloudConfigurationRule{}).
		Named("wizcloudconfigurationrule").
		Complete(r)
}