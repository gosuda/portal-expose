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

package util

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Phase constants for PortalExpose status
const (
	PhaseReady    = "Ready"
	PhasePending  = "Pending"
	PhaseDegraded = "Degraded"
	PhaseFailed   = "Failed"
)

// Condition type constants
const (
	// ConditionAvailable indicates the PortalExpose is Ready or Degraded
	ConditionAvailable = "Available"

	// ConditionProgressing indicates a rolling update is in progress
	ConditionProgressing = "Progressing"

	// ConditionTunnelDeploymentReady indicates all tunnel pods are ready
	ConditionTunnelDeploymentReady = "TunnelDeploymentReady"

	// ConditionRelayConnected indicates all relays are connected
	ConditionRelayConnected = "RelayConnected"

	// ConditionServiceExists indicates the referenced Service was found
	ConditionServiceExists = "ServiceExists"
)

// SetCondition updates or adds a condition to the condition list
// Only updates lastTransitionTime if the status actually changed
func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.NewTime(time.Now())

	// Find existing condition
	for i, existing := range *conditions {
		if existing.Type == conditionType {
			// Only update lastTransitionTime if status changed
			if existing.Status != status {
				(*conditions)[i].LastTransitionTime = now
			}
			(*conditions)[i].Status = status
			(*conditions)[i].Reason = reason
			(*conditions)[i].Message = message
			(*conditions)[i].ObservedGeneration = 0 // Will be set by caller if needed
			return
		}
	}

	// Condition doesn't exist, add it
	*conditions = append(*conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: 0,
	})
}

// FindCondition finds a condition by type
func FindCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

// IsConditionTrue checks if a condition exists and has status True
func IsConditionTrue(conditions []metav1.Condition, conditionType string) bool {
	condition := FindCondition(conditions, conditionType)
	return condition != nil && condition.Status == metav1.ConditionTrue
}
