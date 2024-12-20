package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"kubevirt.io/ssp-operator/internal/common"
	crd_watch "kubevirt.io/ssp-operator/internal/crd-watch"
	"kubevirt.io/ssp-operator/internal/env"
)

// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingadmissionpolicies,verbs=get;list;create;watch;update

const (
	vapControllerName                           = "vap-controller"
	virtualMachineDeleteProtectionAppLabelValue = "kubevirt-vm-deletion-protection"
	virtualMachineDeleteProtectionPolicyName    = "kubevirt-vm-deletion-protection-policy"
)

type vapController struct {
	client client.Client
	log    logr.Logger
}

var _ Controller = &vapController{}

var _ reconcile.Reconciler = &vapController{}

func NewVAPController() Controller {
	return &vapController{
		log: ctrl.Log.WithName("controllers").WithName("VAP"),
	}
}

func (v *vapController) Name() string { return vapControllerName }

func (v *vapController) AddToManager(mgr ctrl.Manager, _ crd_watch.CrdList) error {
	return mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		v.client = mgr.GetClient()

		vap := NewVMDeleteProtectionValidatingAdmissionPolicy()

		err := v.client.Create(ctx, vap)

		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}

		return v.setupController(mgr)
	}))
}

func (v *vapController) setupController(mgr ctrl.Manager) error {
	// Define predicate functions
	predicateVAP := func(object client.Object) bool {
		return object.GetName() == virtualMachineDeleteProtectionPolicyName
	}

	// Combine predicates
	combinedPredicate := predicate.NewPredicateFuncs(predicateVAP)

	return ctrl.NewControllerManagedBy(mgr).
		Named(vapControllerName).
		For(&admissionregistrationv1.ValidatingAdmissionPolicy{}, builder.WithPredicates(combinedPredicate)).
		Complete(v)
}

func (v vapController) RequiredCrds() []string { return nil }

func (v *vapController) Reconcile(ctx context.Context, req ctrl.Request) (reconcile.Result, error) {
	v.log.Info("Starting ValidatingAdmissionPolicy reconciliation...", "request", req.String())

	err := v.reconcileVAP(ctx)

	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, err
}

func (v *vapController) reconcileVAP(ctx context.Context) error {
	vap := NewVMDeleteProtectionValidatingAdmissionPolicy()
	foundVAP := &admissionregistrationv1.ValidatingAdmissionPolicy{}

	objKey := client.ObjectKey{Name: vap.Name}
	err := v.client.Get(ctx, objKey, foundVAP)

	v.log.Info(fmt.Sprintf("CANO: found VAP %v", foundVAP.String()))
	v.log.Info(fmt.Sprintf("CANO: error %v", err))

	if err != nil {
		if errors.IsNotFound(err) {
			err = v.client.Create(ctx, vap)
			if err != nil {
				return fmt.Errorf("failed to create VAP %v", err)
			}
			return err
		}
	}

	if requiresUpdate(vap.Labels, foundVAP.Labels) {
		foundVAP.Labels = vap.Labels
		err = v.client.Update(ctx, foundVAP)
		if err != nil {
			return err
		}
	}

	if requiresUpdate(foundVAP.Spec, vap.Spec) {
		foundVAP.Spec = *vap.Spec.DeepCopy()
		err = v.client.Update(ctx, foundVAP)
		if err != nil {
			return err
		}
	}

	return nil
}

func requiresUpdate[T any](old, new T) bool {
	if !reflect.DeepEqual(old, new) {
		return true
	}
	return false
}

func NewVMDeleteProtectionValidatingAdmissionPolicy() *admissionregistrationv1.ValidatingAdmissionPolicy {
	failPolicy := admissionregistrationv1.Fail

	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: virtualMachineDeleteProtectionPolicyName,
			Labels: map[string]string{
				common.AppKubernetesManagedByLabel: common.AppKubernetesManagedByValue,
				common.AppKubernetesVersionLabel:   env.GetOperatorVersion(),
				common.AppKubernetesComponentLabel: vapControllerName,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicySpec{
			FailurePolicy: &failPolicy,
			MatchConstraints: &admissionregistrationv1.MatchResources{
				ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1.RuleWithOperations{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Delete,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"kubevirt.io"},
								APIVersions: []string{"*"},
								Resources:   []string{"virtualmachines"},
							},
						},
					},
				},
			},
			Variables: []admissionregistrationv1.Variable{
				{
					Name:       "label",
					Expression: `string('kubevirt.io/vm-delete-protection')`,
				},
			},
			Validations: []admissionregistrationv1.Validation{
				{
					Expression:        `(!(variables.label in oldObject.metadata.labels) || !oldObject.metadata.labels[variables.label].matches('^(true|True)$'))`,
					MessageExpression: `'VirtualMachine ' + string(oldObject.metadata.name) + ' cannot be deleted, remove delete protection'`,
				},
			},
		},
	}
}
