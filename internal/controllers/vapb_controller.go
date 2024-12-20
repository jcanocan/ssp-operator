package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/ssp-operator/internal/common"
	crd_watch "kubevirt.io/ssp-operator/internal/crd-watch"
	"kubevirt.io/ssp-operator/internal/env"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingadmissionpolicybindings,verbs=get;list;create;watch;update

const (
	vapbControllerName                              = "vapb-controller"
	virtualMachineDeleteProtectionPolicyBindingName = "kubevirt-vm-deletion-protection-binding"
)

type vapbController struct {
	client client.Client
	log    logr.Logger
}

var _ Controller = &vapbController{}

var _ reconcile.Reconciler = &vapbController{}

func NewVAPBController() Controller {
	return &vapbController{
		log: ctrl.Log.WithName("controllers").WithName("VAPB"),
	}
}

func (v *vapbController) Name() string { return vapbControllerName }

func (v *vapbController) AddToManager(mgr ctrl.Manager, _ crd_watch.CrdList) error {
	return mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		v.client = mgr.GetClient()

		vapb := NewVMDeleteProtectionValidatingAdmissionPolicyBinding()

		err := v.client.Create(ctx, vapb)

		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
		return v.setupController(mgr)
	}))
}

func (v *vapbController) setupController(mgr ctrl.Manager) error {
	// Define predicate functions
	predicateVAPB := func(object client.Object) bool {
		return object.GetName() == virtualMachineDeleteProtectionPolicyBindingName
	}

	// Combine predicates
	combinedPredicate := predicate.NewPredicateFuncs(predicateVAPB)

	return ctrl.NewControllerManagedBy(mgr).
		Named(vapbControllerName).
		For(&admissionregistrationv1.ValidatingAdmissionPolicyBinding{}, builder.WithPredicates(combinedPredicate)).
		Complete(v)
}

func (v vapbController) RequiredCrds() []string { return nil }

func (v *vapbController) Reconcile(ctx context.Context, req ctrl.Request) (reconcile.Result, error) {
	v.log.Info("Starting ValidatingAdmissionPolicyBinding reconciliation...", "request", req.String())

	err := v.reconcileVAPB(ctx)

	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, err
}

func (v *vapbController) reconcileVAPB(ctx context.Context) error {
	vapb := NewVMDeleteProtectionValidatingAdmissionPolicyBinding()
	foundVAPB := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}

	objKey := client.ObjectKey{Name: vapb.Name}
	err := v.client.Get(ctx, objKey, foundVAPB)

	v.log.Info(fmt.Sprintf("CANO: found VAPB %v", foundVAPB.String()))
	v.log.Info(fmt.Sprintf("CANO: error %v", err))

	if err != nil {
		if errors.IsNotFound(err) {
			err = v.client.Create(ctx, vapb)
			if err != nil {
				return fmt.Errorf("failed to create VAP %v", err)
			}
			return err
		}
	}

	if requiresUpdate(vapb.Labels, foundVAPB.Labels) {
		foundVAPB.Labels = vapb.Labels
		err = v.client.Update(ctx, foundVAPB)
		if err != nil {
			return err
		}
	}

	if requiresUpdate(foundVAPB.Spec, vapb.Spec) {
		foundVAPB.Spec = *vapb.Spec.DeepCopy()
		err = v.client.Update(ctx, foundVAPB)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewVMDeleteProtectionValidatingAdmissionPolicyBinding() *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: virtualMachineDeleteProtectionPolicyBindingName,
			Labels: map[string]string{
				common.AppKubernetesManagedByLabel: common.AppKubernetesManagedByValue,
				common.AppKubernetesVersionLabel:   env.GetOperatorVersion(),
				common.AppKubernetesComponentLabel: vapbControllerName,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: virtualMachineDeleteProtectionPolicyName,
			ValidationActions: []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Deny,
			},
		},
	}
}
