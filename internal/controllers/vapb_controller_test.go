package controllers

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"kubevirt.io/ssp-operator/internal/common"
)

var _ = Describe("VAPB Controller", func() {
	var (
		fakeClient         client.Client
		testRequest        reconcile.Request
		testVAPBController *vapbController
		testVAPBPolicy     *admissionregistrationv1.ValidatingAdmissionPolicyBinding
	)

	Context("reconcile methods", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(common.Scheme).Build()
			testVAPBController = NewVAPBController().(*vapbController)
			testVAPBPolicy = NewVMDeleteProtectionValidatingAdmissionPolicyBinding()
			testVAPBController.client = fakeClient
			testRequest = reconcile.Request{NamespacedName: types.NamespacedName{
				Name: virtualMachineDeleteProtectionPolicyBindingName,
			}}
		})

		It("should create the policy it is not present", func() {
			_, err := testVAPBController.Reconcile(context.Background(), testRequest)
			Expect(err).ToNot(HaveOccurred())

			foundVAPB := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
			objKey := client.ObjectKey{Name: virtualMachineDeleteProtectionPolicyBindingName}

			err = fakeClient.Get(context.Background(), objKey, foundVAPB)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundVAPB.Name).To(Equal(testVAPBPolicy.Name))
			Expect(foundVAPB.Labels).To(Equal(testVAPBPolicy.Labels))
			Expect(foundVAPB.Spec).To(Equal(testVAPBPolicy.Spec))
		})

		It("should update labels if changed", func() {
			modifiedVAPB := NewVMDeleteProtectionValidatingAdmissionPolicyBinding()
			modifiedVAPB.Labels["test-label"] = "test-label"
			err := fakeClient.Create(context.Background(), modifiedVAPB)
			Expect(err).ToNot(HaveOccurred())

			_, err = testVAPBController.Reconcile(context.Background(), testRequest)
			Expect(err).ToNot(HaveOccurred())

			foundVAPB := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
			objKey := client.ObjectKey{Name: virtualMachineDeleteProtectionPolicyBindingName}
			err = fakeClient.Get(context.Background(), objKey, foundVAPB)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundVAPB.Name).To(Equal(testVAPBPolicy.Name))
			Expect(foundVAPB.Labels).To(Equal(testVAPBPolicy.Labels))
			Expect(foundVAPB.Spec).To(Equal(testVAPBPolicy.Spec))
		})

		It("should update spec fi changed", func() {
			modifiedVAPB := NewVMDeleteProtectionValidatingAdmissionPolicyBinding()
			modifiedVAPB.Spec.ValidationActions = []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Warn,
			}
			err := fakeClient.Create(context.Background(), modifiedVAPB)
			Expect(err).ToNot(HaveOccurred())

			_, err = testVAPBController.Reconcile(context.Background(), testRequest)
			Expect(err).ToNot(HaveOccurred())

			foundVAPB := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
			objKey := client.ObjectKey{Name: virtualMachineDeleteProtectionPolicyBindingName}
			err = fakeClient.Get(context.Background(), objKey, foundVAPB)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundVAPB.Name).To(Equal(testVAPBPolicy.Name))
			Expect(foundVAPB.Labels).To(Equal(testVAPBPolicy.Labels))
			Expect(foundVAPB.Spec).To(Equal(testVAPBPolicy.Spec))
		})
	})
})
