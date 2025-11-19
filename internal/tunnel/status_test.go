package tunnel

import (
	"testing"

	portalv1alpha1 "github.com/gosuda/portal-expose/api/v1alpha1"
)

func TestConstructPublicURL(t *testing.T) {
	tests := []struct {
		name     string
		appName  string
		relayURL string
		want     string
	}{
		{
			name:     "Standard WSS URL",
			appName:  "my-app",
			relayURL: "wss://portal.gosuda.org/relay",
			want:     "https://my-app.portal.gosuda.org",
		},
		{
			name:     "WSS URL with port",
			appName:  "test",
			relayURL: "wss://relay.example.com:8443/connect",
			want:     "https://test.relay.example.com:8443",
		},
		{
			name:     "Simple domain",
			appName:  "demo",
			relayURL: "wss://simple.com",
			want:     "https://demo.simple.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConstructPublicURL(tt.appName, tt.relayURL); got != tt.want {
				t.Errorf("ConstructPublicURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputePhase(t *testing.T) {
	tests := []struct {
		name           string
		readyPods      int32
		totalPods      int32
		relayConnected int
		totalRelays    int
		want           string
	}{
		{
			name:           "All Ready",
			readyPods:      1,
			totalPods:      1,
			relayConnected: 1,
			totalRelays:    1,
			want:           "Ready",
		},
		{
			name:           "Pending (No pods ready)",
			readyPods:      0,
			totalPods:      1,
			relayConnected: 0,
			totalRelays:    1,
			want:           "Pending",
		},
		{
			name:           "Degraded (Pod ready, Relay disconnected)",
			readyPods:      1,
			totalPods:      1,
			relayConnected: 0,
			totalRelays:    1,
			want:           "Degraded",
		},
		{
			name:           "Degraded (Pod not ready, Relay connected - theoretical)",
			readyPods:      0,
			totalPods:      1,
			relayConnected: 1,
			totalRelays:    1,
			want:           "Degraded",
		},
		{
			name:           "Failed (Nothing working)",
			readyPods:      0,
			totalPods:      0, // Or 0 ready/1 total and 0 relays
			relayConnected: 0,
			totalRelays:    0,
			want:           "Failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputePhase(tt.readyPods, tt.totalPods, tt.relayConnected, tt.totalRelays); got != tt.want {
				t.Errorf("ComputePhase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeRelayStatuses(t *testing.T) {
	targets := []portalv1alpha1.RelayTarget{
		{Name: "relay1"},
		{Name: "relay2"},
	}

	tests := []struct {
		name      string
		podsReady bool
		wantState string
	}{
		{
			name:      "Pods Ready -> Connected",
			podsReady: true,
			wantState: "Connected",
		},
		{
			name:      "Pods Not Ready -> Disconnected",
			podsReady: false,
			wantState: "Disconnected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statuses := ComputeRelayStatuses(targets, tt.podsReady)
			if len(statuses) != 2 {
				t.Errorf("Expected 2 statuses, got %d", len(statuses))
			}
			for _, s := range statuses {
				if s.Status != tt.wantState {
					t.Errorf("Status = %v, want %v", s.Status, tt.wantState)
				}
			}
		})
	}
}
