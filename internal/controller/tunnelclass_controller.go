/*
Copyright 2025.

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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	portalv1alpha1 "github.com/gosuda/portal-expose/api/v1alpha1"
)

// TunnelClassReconciler reconciles a TunnelClass object
type TunnelClassReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=portal.gosuda.org,resources=tunnelclasses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=portal.gosuda.org,resources=tunnelclasses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=portal.gosuda.org,resources=tunnelclasses/finalizers,verbs=update

// Reconcile ensures TunnelClass consistency, particularly around default class handling
func (r *TunnelClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the TunnelClass
	tunnelClass := &portalv1alpha1.TunnelClass{}
	if err := r.Get(ctx, req.NamespacedName, tunnelClass); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "unable to fetch TunnelClass")
			return ctrl.Result{}, err
		}
		// Resource deleted - nothing to do
		return ctrl.Result{}, nil
	}

	// Check if this TunnelClass is marked as default
	isDefault := tunnelClass.Annotations != nil &&
		tunnelClass.Annotations["portal.gosuda.org/is-default-class"] == "true"

	if isDefault {
		// Ensure only one TunnelClass is default
		if err := r.ensureOnlyOneDefault(ctx, tunnelClass); err != nil {
			log.Error(err, "failed to ensure only one default TunnelClass")
			return ctrl.Result{}, err
		}
	}

	log.V(1).Info("TunnelClass reconciled", "name", tunnelClass.Name, "isDefault", isDefault)
	return ctrl.Result{}, nil
}

// ensureOnlyOneDefault removes the default annotation from other TunnelClasses
func (r *TunnelClassReconciler) ensureOnlyOneDefault(ctx context.Context, newDefault *portalv1alpha1.TunnelClass) error {
	log := logf.FromContext(ctx)

	// List all TunnelClasses
	tunnelClasses := &portalv1alpha1.TunnelClassList{}
	if err := r.List(ctx, tunnelClasses); err != nil {
		return err
	}

	// Remove default annotation from other TunnelClasses
	for i := range tunnelClasses.Items {
		tc := &tunnelClasses.Items[i]

		// Skip the new default
		if tc.Name == newDefault.Name {
			continue
		}

		// Check if this one is also marked as default
		if tc.Annotations != nil && tc.Annotations["portal.gosuda.org/is-default-class"] == "true" {
			log.Info("Removing default annotation from previous default TunnelClass",
				"name", tc.Name, "newDefault", newDefault.Name)

			// Remove the annotation
			delete(tc.Annotations, "portal.gosuda.org/is-default-class")

			if err := r.Update(ctx, tc); err != nil {
				return err
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TunnelClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&portalv1alpha1.TunnelClass{}).
		Named("tunnelclass").
		Complete(r)
}
