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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	securityv1 "github.com/aldarisbm/wiz-ccr/api/v1"
	"github.com/aldarisbm/wiz-ccr/internal/wiz"
)

// fakeWizClient is a no-op Wiz client for use in tests.
type fakeWizClient struct {
	createdID string
}

func (f *fakeWizClient) CreateRule(_ context.Context, _ wiz.Rule) (string, error) {
	return f.createdID, nil
}

func (f *fakeWizClient) UpdateRule(_ context.Context, _ string, _ wiz.Rule) error {
	return nil
}

func (f *fakeWizClient) DeleteRule(_ context.Context, _ string) error {
	return nil
}

var _ = Describe("WizCloudConfigurationRule Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		wizcloudconfigurationrule := &securityv1.WizCloudConfigurationRule{}

		ruleName := "Test rule"
		code := "package main\ndeny[msg] { msg := \"denied\" }"

		BeforeEach(func() {
			By("creating the custom resource for the Kind WizCloudConfigurationRule")
			err := k8sClient.Get(ctx, typeNamespacedName, wizcloudconfigurationrule)
			if err != nil && errors.IsNotFound(err) {
				resource := &securityv1.WizCloudConfigurationRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: securityv1.WizCloudConfigurationRuleSpec{
						RuleName:         &ruleName,
						TargetNativeType: strPtr("Pod"),
						Matchers:         []securityv1.MatcherType{securityv1.MatcherTypeAdmissionsController},
						OperationTypes:   []securityv1.OperationType{securityv1.Create},
						Code:             &code,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &securityv1.WizCloudConfigurationRule{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance WizCloudConfigurationRule")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &WizCloudConfigurationRuleReconciler{
				Client:    k8sClient,
				Scheme:    k8sClient.Scheme(),
				WizClient: &fakeWizClient{createdID: "wiz-rule-123"},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the Wiz rule ID was stored in status")
			updated := &securityv1.WizCloudConfigurationRule{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, updated)).To(Succeed())
			Expect(updated.Status.WizRuleID).To(Equal("wiz-rule-123"))

			By("Checking that the Available condition is set")
			Expect(updated.Status.Conditions).To(ContainElement(
				HaveField("Type", "Available"),
			))
		})
	})
})

func strPtr(s string) *string { return &s }
