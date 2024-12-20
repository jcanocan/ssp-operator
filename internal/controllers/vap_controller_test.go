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

var _ = Describe("VAP Controller", func() {
	var (
		fakeClient        client.Client
		testRequest       reconcile.Request
		testVAPController *vapController
		testVAPPolicy     *admissionregistrationv1.ValidatingAdmissionPolicy
	)

	Context("reconcile methods", func() {
		BeforeEach(func() {
			fakeClient = fake.NewClientBuilder().WithScheme(common.Scheme).Build()
			testVAPController = NewVAPController().(*vapController)
			testVAPPolicy = NewVMDeleteProtectionValidatingAdmissionPolicy()
			testVAPController.client = fakeClient
			testRequest = reconcile.Request{NamespacedName: types.NamespacedName{
				Name: virtualMachineDeleteProtectionPolicyName,
			}}
		})

		It("should create the policy it is not present", func() {
			_, err := testVAPController.Reconcile(context.Background(), testRequest)
			Expect(err).ToNot(HaveOccurred())

			foundVAP := &admissionregistrationv1.ValidatingAdmissionPolicy{}
			objKey := client.ObjectKey{Name: virtualMachineDeleteProtectionPolicyName}

			err = fakeClient.Get(context.Background(), objKey, foundVAP)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundVAP.Name).To(Equal(testVAPPolicy.Name))
			Expect(foundVAP.Labels).To(Equal(testVAPPolicy.Labels))
			Expect(foundVAP.Spec).To(Equal(testVAPPolicy.Spec))
		})

		It("should update labels if changed", func() {
			modifiedVAP := NewVMDeleteProtectionValidatingAdmissionPolicy()
			modifiedVAP.Labels["test-label"] = "test-label"
			err := fakeClient.Create(context.Background(), modifiedVAP)
			Expect(err).ToNot(HaveOccurred())

			_, err = testVAPController.Reconcile(context.Background(), testRequest)
			Expect(err).ToNot(HaveOccurred())

			foundVAP := &admissionregistrationv1.ValidatingAdmissionPolicy{}
			objKey := client.ObjectKey{Name: virtualMachineDeleteProtectionPolicyName}
			err = fakeClient.Get(context.Background(), objKey, foundVAP)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundVAP.Name).To(Equal(testVAPPolicy.Name))
			Expect(foundVAP.Labels).To(Equal(testVAPPolicy.Labels))
			Expect(foundVAP.Spec).To(Equal(testVAPPolicy.Spec))
		})

		It("should update spec fi changed", func() {
			modifiedVAP := NewVMDeleteProtectionValidatingAdmissionPolicy()
			modifiedVAP.Spec.Variables = []admissionregistrationv1.Variable{
				{
					Name:       "test-variable",
					Expression: `test-expression`,
				},
			}
			err := fakeClient.Create(context.Background(), modifiedVAP)
			Expect(err).ToNot(HaveOccurred())

			_, err = testVAPController.Reconcile(context.Background(), testRequest)
			Expect(err).ToNot(HaveOccurred())

			foundVAP := &admissionregistrationv1.ValidatingAdmissionPolicy{}
			objKey := client.ObjectKey{Name: virtualMachineDeleteProtectionPolicyName}
			err = fakeClient.Get(context.Background(), objKey, foundVAP)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundVAP.Name).To(Equal(testVAPPolicy.Name))
			Expect(foundVAP.Labels).To(Equal(testVAPPolicy.Labels))
			Expect(foundVAP.Spec).To(Equal(testVAPPolicy.Spec))
		})
	})
})
