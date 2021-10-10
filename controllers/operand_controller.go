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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mathv1 "a-ksy/math-operator/api/v1"
)

// OperandReconciler reconciles a Operand object
type OperandReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=math.a-ksy,resources=operands,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=math.a-ksy,resources=operands/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=math.a-ksy,resources=operands/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Operand object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *OperandReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("operand", req.NamespacedName)

	log.Info("Reconciliation started for Operand")

	// Fetching Operator
	operator := &mathv1.Operator{}

	log.Info("Fetching Operator")
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: "default"}, operator); err != nil {
		log.Error(err, "Unable to retrieve Operator")

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Succesfuly fetched Operator")

	// Fetching Operand
	operand := &mathv1.Operand{}

	log.Info("Fetching Operand")
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, operand); err != nil {
		log.Error(err, "Unable to retrieve Operand")

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Succesfuly fetched Operand")

	// Fetching Result

	// Fetching Operand
	result := &mathv1.Result{}

	log.Info("Fetching Result")
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: "default"}, result); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Result doesn't exist, creating one")

			result = &mathv1.Result{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: req.Namespace,
				},
				Status: mathv1.ResultStatus{
					Result:      0,
					Calculation: "",
				},
			}

			if err := r.Client.Create(ctx, result); err != nil {
				log.Error(err, "Unable to create Result")
				return ctrl.Result{}, err
			}
			log.Info("Succesfuly created Result")
		} else {
			log.Error(err, "Unable to retrieve Result")
			return ctrl.Result{}, err
		}
	}
	log.Info("Succesfuly fetched Result")

	// If Clear is true, set the result to operator.Value; otherwise calculate the new result with respect to the operator
	if operand.Spec.Clear {
		if val, err := calculate(operator.Spec.Operation, operand.Spec.Value, 0); err != nil {
			log.Error(err, "Unable to calculate the value of Result")
			return ctrl.Result{}, err
		} else {
			result.Status.Result = val
		}
		result.Status.Calculation = fmt.Sprintf("%d", operand.Spec.Value)
	} else {
		if val, err := calculate(operator.Spec.Operation, operand.Spec.Value, result.Status.Result); err != nil {
			log.Error(err, "Unable the calculate the value of Result")

			return ctrl.Result{}, err
		} else {
			result.Status.Result = val

			if calculation, err := generateCalculationString(result.Status.Calculation, operator.Spec.Operation, operand.Spec.Value); err != nil {
				log.Error(err, "Unable to generate Calculation string")

				return ctrl.Result{}, err
			} else {
				result.Status.Calculation = calculation
			}
		}
	}

	log.Info("Updating Result")
	if err := r.Client.Status().Update(ctx, result); err != nil {
		log.Error(err, "Unable to update Result")

		return ctrl.Result{}, err
	}
	log.Info("Succesfuly updated Result")
	log.Info(fmt.Sprintf("New Result is: %d", result.Status.Result))

	log.Info("Reconciliation ended for Operand")
	return ctrl.Result{}, nil
}

func calculate(operation mathv1.Operation, value int64, oldResult int64) (int64, error) {

	switch operation {
	case mathv1.Addition:
		return oldResult + value, nil
	case mathv1.Subtraction:
		return oldResult - value, nil
	case mathv1.Multiplication:
		return oldResult * value, nil
	case mathv1.Division:
		if value == 0 {
			return -1, errors.NewBadRequest("Can't divide by zero")
		}
		return oldResult / value, nil
	}

	return -1, errors.NewBadRequest("Operation is invalid")
}

func generateCalculationString(prevCalculation string, operation mathv1.Operation, value int64) (string, error) {

	switch operation {
	case mathv1.Addition:
		return fmt.Sprintf("%s + %d", prevCalculation, value), nil
	case mathv1.Subtraction:
		return fmt.Sprintf("%s - %d", prevCalculation, value), nil
	case mathv1.Multiplication:
		return fmt.Sprintf("(%s) * %d", prevCalculation, value), nil
	case mathv1.Division:
		return fmt.Sprintf("(%s) / %d", prevCalculation, value), nil
	}

	return "", errors.NewBadRequest("Operation is invalid")
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperandReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mathv1.Operand{}).
		Complete(r)
}
