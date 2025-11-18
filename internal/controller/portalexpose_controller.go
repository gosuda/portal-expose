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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	portalv1alpha1 "github.com/gosuda/portal-expose/api/v1alpha1"
	"github.com/gosuda/portal-expose/internal/tunnel"
	"github.com/gosuda/portal-expose/internal/tunnelclass"
	"github.com/gosuda/portal-expose/internal/util"
)

// PortalExposeReconciler reconciles a PortalExpose object
type PortalExposeReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=portal.gosuda.org,resources=portalexposes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=portal.gosuda.org,resources=portalexposes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=portal.gosuda.org,resources=portalexposes/finalizers,verbs=update
// +kubebuilder:rbac:groups=portal.gosuda.org,resources=tunnelclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PortalExposeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling PortalExpose", "name", req.Name, "namespace", req.Namespace)

	// Fetch the PortalExpose instance
	portalExpose := &portalv1alpha1.PortalExpose{}
	if err := r.Get(ctx, req.NamespacedName, portalExpose); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted after reconcile request
			logger.Info("PortalExpose resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get PortalExpose")
		return ctrl.Result{}, err
	}

	// 1. Handle deletion
	if !portalExpose.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, portalExpose)
	}

	// 2. Add finalizer if missing
	if !util.HasFinalizer(portalExpose, util.FinalizerName) {
		util.AddFinalizer(portalExpose, util.FinalizerName)
		if err := r.Update(ctx, portalExpose); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		logger.Info("Added finalizer")
		return ctrl.Result{Requeue: true}, nil
	}

	// 3. Validate referenced Service exists
	service := &corev1.Service{}
	serviceKey := types.NamespacedName{
		Name:      portalExpose.Spec.App.Service.Name,
		Namespace: portalExpose.Namespace,
	}
	if err := r.Get(ctx, serviceKey, service); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Service not found", "service", portalExpose.Spec.App.Service.Name)
			portalExpose.Status.Phase = "Failed"
			util.SetCondition(&portalExpose.Status.Conditions, util.ConditionServiceExists, metav1.ConditionFalse,
				"ServiceNotFound", fmt.Sprintf("Service '%s' not found in namespace '%s'", portalExpose.Spec.App.Service.Name, portalExpose.Namespace))
			util.SetCondition(&portalExpose.Status.Conditions, util.ConditionAvailable, metav1.ConditionFalse,
				"ServiceNotFound", "PortalExpose failed due to missing Service")

			r.Recorder.Event(portalExpose, corev1.EventTypeWarning, "ServiceNotFound",
				fmt.Sprintf("Referenced Service '%s' not found", portalExpose.Spec.App.Service.Name))

			if err := r.Status().Update(ctx, portalExpose); err != nil {
				logger.Error(err, "Failed to update status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil // Don't requeue, wait for Service creation event
		}
		logger.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	}

	util.SetCondition(&portalExpose.Status.Conditions, util.ConditionServiceExists, metav1.ConditionTrue,
		"ServiceFound", "Service exists")

	// 4. Resolve TunnelClass
	tunnelClass, err := r.resolveTunnelClass(ctx, portalExpose)
	if err != nil {
		logger.Error(err, "Failed to resolve TunnelClass")
		portalExpose.Status.Phase = "Failed"
		util.SetCondition(&portalExpose.Status.Conditions, "TunnelClassExists", metav1.ConditionFalse,
			"TunnelClassNotFound", err.Error())

		r.Recorder.Event(portalExpose, corev1.EventTypeWarning, "TunnelClassNotFound", err.Error())

		if statusErr := r.Status().Update(ctx, portalExpose); statusErr != nil {
			logger.Error(statusErr, "Failed to update status")
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}

	// 5. Generate desired Deployment spec
	desiredDeployment := tunnel.BuildDeployment(portalExpose, tunnelClass)

	// Set PortalExpose as owner of the Deployment
	if err := controllerutil.SetControllerReference(portalExpose, desiredDeployment, r.Scheme); err != nil {
		logger.Error(err, "Failed to set controller reference")
		return ctrl.Result{}, err
	}

	// 6. Reconcile Deployment
	existingDeployment := &appsv1.Deployment{}
	deploymentKey := types.NamespacedName{
		Name:      desiredDeployment.Name,
		Namespace: desiredDeployment.Namespace,
	}
	err = r.Get(ctx, deploymentKey, existingDeployment)
	if err != nil && errors.IsNotFound(err) {
		// Deployment doesn't exist, create it
		logger.Info("Creating tunnel Deployment", "name", desiredDeployment.Name)
		if err := r.Create(ctx, desiredDeployment); err != nil {
			logger.Error(err, "Failed to create Deployment")
			return ctrl.Result{}, err
		}

		portalExpose.Status.Phase = "Pending"
		util.SetCondition(&portalExpose.Status.Conditions, util.ConditionProgressing, metav1.ConditionTrue,
			"DeploymentCreated", "Tunnel Deployment created, waiting for pods")

		r.Recorder.Event(portalExpose, corev1.EventTypeNormal, "Created",
			"PortalExpose created, deploying tunnel pods")

		// Construct public URL
		primaryRelayURL := portalExpose.Spec.Relay.Targets[0].URL
		portalExpose.Status.PublicURL = tunnel.ConstructPublicURL(portalExpose.Spec.App.Name, primaryRelayURL)

		if err := r.Status().Update(ctx, portalExpose); err != nil {
			logger.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil // Requeue to check pod readiness
	} else if err != nil {
		logger.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Deployment exists, check if update needed
	// For MVP, we'll do a simple comparison of spec fields
	// In production, use a more sophisticated comparison
	if !deploymentSpecEqual(existingDeployment, desiredDeployment) {
		logger.Info("Updating tunnel Deployment", "name", existingDeployment.Name)
		existingDeployment.Spec = desiredDeployment.Spec
		if err := r.Update(ctx, existingDeployment); err != nil {
			logger.Error(err, "Failed to update Deployment")
			return ctrl.Result{}, err
		}

		util.SetCondition(&portalExpose.Status.Conditions, util.ConditionProgressing, metav1.ConditionTrue,
			"DeploymentUpdating", "Rolling update in progress")

		if err := r.Status().Update(ctx, portalExpose); err != nil {
			logger.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 7. Compute status from Deployment
	readyReplicas := existingDeployment.Status.ReadyReplicas
	desiredReplicas := tunnelClass.Spec.Replicas

	portalExpose.Status.TunnelPods.Ready = readyReplicas
	portalExpose.Status.TunnelPods.Total = desiredReplicas

	// 8. Compute relay connection status (simplified for MVP)
	podsReady := (readyReplicas > 0)
	relayStatuses := tunnel.ComputeRelayStatuses(portalExpose.Spec.Relay.Targets, podsReady)
	portalExpose.Status.Relay.Connected = relayStatuses

	// 9. Compute phase
	connectedRelays := 0
	for _, rs := range relayStatuses {
		if rs.Status == "Connected" {
			connectedRelays++
		}
	}
	portalExpose.Status.Phase = tunnel.ComputePhase(readyReplicas, desiredReplicas, connectedRelays, len(relayStatuses))

	// 10. Ensure public URL is set
	if portalExpose.Status.PublicURL == "" {
		primaryRelayURL := portalExpose.Spec.Relay.Targets[0].URL
		portalExpose.Status.PublicURL = tunnel.ConstructPublicURL(portalExpose.Spec.App.Name, primaryRelayURL)
	}

	// 11. Update conditions
	allPodsReady := (readyReplicas == desiredReplicas && desiredReplicas > 0)
	allRelaysConnected := (connectedRelays == len(relayStatuses) && len(relayStatuses) > 0)

	if allPodsReady {
		util.SetCondition(&portalExpose.Status.Conditions, util.ConditionTunnelDeploymentReady, metav1.ConditionTrue,
			"AllPodsReady", fmt.Sprintf("%d/%d tunnel pods ready", readyReplicas, desiredReplicas))
	} else {
		util.SetCondition(&portalExpose.Status.Conditions, util.ConditionTunnelDeploymentReady, metav1.ConditionFalse,
			"PodsNotReady", fmt.Sprintf("Only %d/%d tunnel pods ready", readyReplicas, desiredReplicas))
	}

	if allRelaysConnected {
		util.SetCondition(&portalExpose.Status.Conditions, util.ConditionRelayConnected, metav1.ConditionTrue,
			"AllRelaysConnected", fmt.Sprintf("Connected to %d/%d relays", connectedRelays, len(relayStatuses)))
	} else {
		util.SetCondition(&portalExpose.Status.Conditions, util.ConditionRelayConnected, metav1.ConditionFalse,
			"PartialRelayConnection", fmt.Sprintf("Only %d/%d relays connected", connectedRelays, len(relayStatuses)))
	}

	if portalExpose.Status.Phase == "Ready" || portalExpose.Status.Phase == "Degraded" {
		util.SetCondition(&portalExpose.Status.Conditions, util.ConditionAvailable, metav1.ConditionTrue,
			"PortalExposeAvailable", "PortalExpose is available")
	} else {
		util.SetCondition(&portalExpose.Status.Conditions, util.ConditionAvailable, metav1.ConditionFalse,
			"PortalExposeNotAvailable", fmt.Sprintf("PortalExpose is %s", portalExpose.Status.Phase))
	}

	updating := (existingDeployment.Status.UpdatedReplicas < desiredReplicas)
	if updating {
		util.SetCondition(&portalExpose.Status.Conditions, util.ConditionProgressing, metav1.ConditionTrue,
			"RollingUpdate", "Deployment rolling update in progress")
	} else {
		util.SetCondition(&portalExpose.Status.Conditions, util.ConditionProgressing, metav1.ConditionFalse,
			"DeploymentStable", "No rolling update in progress")
	}

	// 12. Update status
	if err := r.Status().Update(ctx, portalExpose); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// 13. Emit events for phase changes
	// Note: In production, we'd track previous phase to only emit on changes
	if portalExpose.Status.Phase == "Ready" {
		r.Recorder.Event(portalExpose, corev1.EventTypeNormal, "Ready",
			"All tunnel pods and relays are healthy")
	} else if portalExpose.Status.Phase == "Degraded" {
		r.Recorder.Event(portalExpose, corev1.EventTypeWarning, "Degraded",
			fmt.Sprintf("Partial failure: %d/%d pods ready, %d/%d relays connected",
				readyReplicas, desiredReplicas, connectedRelays, len(relayStatuses)))
	} else if portalExpose.Status.Phase == "Failed" {
		r.Recorder.Event(portalExpose, corev1.EventTypeWarning, "Failed",
			"All tunnel pods or relays unavailable")
	}

	logger.Info("Reconciliation complete", "phase", portalExpose.Status.Phase)
	return ctrl.Result{}, nil
}

// handleDeletion handles cleanup when PortalExpose is being deleted
func (r *PortalExposeReconciler) handleDeletion(ctx context.Context, portalExpose *portalv1alpha1.PortalExpose) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling PortalExpose deletion")

	if !util.HasFinalizer(portalExpose, util.FinalizerName) {
		// Finalizer already removed, nothing to do
		return ctrl.Result{}, nil
	}

	// Delete tunnel Deployment
	deployment := &appsv1.Deployment{}
	deploymentKey := types.NamespacedName{
		Name:      portalExpose.Name + "-tunnel",
		Namespace: portalExpose.Namespace,
	}

	err := r.Get(ctx, deploymentKey, deployment)
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "Failed to get Deployment for deletion")
		return ctrl.Result{}, err
	}

	if err == nil {
		// Deployment exists, delete it
		logger.Info("Deleting tunnel Deployment", "name", deployment.Name)
		if err := r.Delete(ctx, deployment); err != nil {
			logger.Error(err, "Failed to delete Deployment")
			return ctrl.Result{}, err
		}
		// Requeue to verify deletion
		return ctrl.Result{Requeue: true}, nil
	}

	// Deployment is deleted or doesn't exist, remove finalizer
	util.RemoveFinalizer(portalExpose, util.FinalizerName)
	if err := r.Update(ctx, portalExpose); err != nil {
		logger.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	r.Recorder.Event(portalExpose, corev1.EventTypeNormal, "Deleted",
		"PortalExpose deleted, tunnel pods cleaned up")

	logger.Info("PortalExpose deletion complete")
	return ctrl.Result{}, nil
}

// resolveTunnelClass finds the TunnelClass to use (specified or default)
func (r *PortalExposeReconciler) resolveTunnelClass(ctx context.Context, portalExpose *portalv1alpha1.PortalExpose) (*portalv1alpha1.TunnelClass, error) {
	return tunnelclass.GetTunnelClass(ctx, r.Client, portalExpose.Spec.TunnelClassName)
}

// deploymentSpecEqual checks if two Deployment specs are equal
// Simplified comparison for MVP
func deploymentSpecEqual(existing, desired *appsv1.Deployment) bool {
	// Compare key fields that trigger updates
	if *existing.Spec.Replicas != *desired.Spec.Replicas {
		return false
	}

	// Compare container image and resources
	if len(existing.Spec.Template.Spec.Containers) != len(desired.Spec.Template.Spec.Containers) {
		return false
	}

	for i := range existing.Spec.Template.Spec.Containers {
		existingContainer := existing.Spec.Template.Spec.Containers[i]
		desiredContainer := desired.Spec.Template.Spec.Containers[i]

		if existingContainer.Image != desiredContainer.Image {
			return false
		}

		// Compare resources (simplified)
		if !existingContainer.Resources.Requests.Cpu().Equal(*desiredContainer.Resources.Requests.Cpu()) {
			return false
		}
		if !existingContainer.Resources.Requests.Memory().Equal(*desiredContainer.Resources.Requests.Memory()) {
			return false
		}
	}

	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *PortalExposeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&portalv1alpha1.PortalExpose{}).
		Owns(&appsv1.Deployment{}). // Watch Deployments owned by PortalExpose
		Named("portalexpose").
		Complete(r)
}
