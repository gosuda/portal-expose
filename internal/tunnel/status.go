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

package tunnel

import (
	"regexp"
	"strings"

	portalv1alpha1 "github.com/gosuda/portal-expose/api/v1alpha1"
	"github.com/gosuda/portal-expose/internal/util"
)

// ConstructPublicURL builds the public URL from app name and relay domain
// Extracts domain from relay WSS URL like "wss://portal.gosuda.org/relay" -> "portal.gosuda.org"
// Returns "https://{app-name}.{relay-domain}"
func ConstructPublicURL(appName string, relayURL string) string {
	// Extract domain from WSS URL
	// Pattern: wss://{domain}/{path}
	re := regexp.MustCompile(`^wss://([^/]+)`)
	matches := re.FindStringSubmatch(relayURL)

	if len(matches) < 2 {
		// Fallback if regex fails
		domain := strings.TrimPrefix(relayURL, "wss://")
		domain = strings.Split(domain, "/")[0]
		return "https://" + appName + "." + domain
	}

	domain := matches[1]
	return "https://" + appName + "." + domain
}

// ComputePhase determines the phase based on pod readiness and relay connectivity
// Phases: Pending | Ready | Degraded | Failed
func ComputePhase(readyPods, totalPods int32, relayConnected, totalRelays int) string {
	allPodsReady := (readyPods == totalPods && totalPods > 0)
	somePodsReady := (readyPods > 0)
	allRelaysConnected := (relayConnected == totalRelays && totalRelays > 0)
	someRelaysConnected := (relayConnected > 0)

	// Ready: all pods ready AND all relays connected
	if allPodsReady && allRelaysConnected {
		return util.PhaseReady
	}

	// Degraded: some pods ready OR some relays connected
	if somePodsReady || someRelaysConnected {
		return util.PhaseDegraded
	}

	// Pending: initial state, waiting for pods/relays
	// Also used when tunnel Deployment is being created
	if readyPods == 0 && totalPods > 0 {
		return util.PhasePending
	}

	// Failed: no pods ready OR no relays connected
	// Also Failed if Service not found (checked by caller)
	return util.PhaseFailed
}

// ComputeRelayStatuses generates relay connection statuses
// In MVP, this is simplified - assumes Connected if pods are ready
// Real implementation would read status from tunnel pod annotations/logs
func ComputeRelayStatuses(relayTargets []portalv1alpha1.RelayTarget, podsReady bool) []portalv1alpha1.RelayConnectionStatus {
	statuses := make([]portalv1alpha1.RelayConnectionStatus, 0, len(relayTargets))

	for _, target := range relayTargets {
		status := portalv1alpha1.RelayConnectionStatus{
			Name: target.Name,
		}

		if podsReady {
			status.Status = "Connected"
			// ConnectedAt would be set here in real implementation
		} else {
			status.Status = "Disconnected"
		}

		statuses = append(statuses, status)
	}

	return statuses
}
