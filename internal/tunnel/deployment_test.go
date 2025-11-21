package tunnel

import (
	"testing"

	portalv1alpha1 "github.com/gosuda/portal-expose/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildDeployment(t *testing.T) {
	tests := []struct {
		name             string
		portalExpose     *portalv1alpha1.PortalExpose
		tunnelClass      *portalv1alpha1.TunnelClass
		expectedImage    string
		expectedReplicas int32
	}{
		{
			name: "Default configuration",
			portalExpose: &portalv1alpha1.PortalExpose{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: portalv1alpha1.PortalExposeSpec{
					App: portalv1alpha1.AppSpec{
						Name:    "test-app",
						Service: portalv1alpha1.ServiceRef{Name: "test-svc", Port: 80},
					},
					Relay: portalv1alpha1.RelaySpec{
						Targets: []portalv1alpha1.RelayTarget{
							{Name: "relay", URL: "wss://relay.example.com"},
						},
					},
				},
			},
			tunnelClass: &portalv1alpha1.TunnelClass{
				Spec: portalv1alpha1.TunnelClassSpec{
					Replicas: 1,
					Size:     "small",
				},
			},
			expectedImage:    "ghcr.io/gosuda/portal-tunnel:1.0.0",
			expectedReplicas: 1,
		},
		{
			name: "With TunnelClass",
			portalExpose: &portalv1alpha1.PortalExpose{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: portalv1alpha1.PortalExposeSpec{
					App: portalv1alpha1.AppSpec{
						Name:    "test-app",
						Service: portalv1alpha1.ServiceRef{Name: "test-svc", Port: 80},
					},
					Relay: portalv1alpha1.RelaySpec{
						Targets: []portalv1alpha1.RelayTarget{
							{Name: "relay", URL: "wss://relay.example.com"},
						},
					},
				},
			},
			tunnelClass: &portalv1alpha1.TunnelClass{
				Spec: portalv1alpha1.TunnelClassSpec{
					Replicas: 3,
				},
			},
			expectedImage:    "ghcr.io/gosuda/portal-tunnel:1.0.0",
			expectedReplicas: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := BuildDeployment(tt.portalExpose, tt.tunnelClass)

			if deployment.Name != tt.portalExpose.Name+"-tunnel" {
				t.Errorf("BuildDeployment() name = %v, want %v", deployment.Name, tt.portalExpose.Name+"-tunnel")
			}

			if *deployment.Spec.Replicas != tt.expectedReplicas {
				t.Errorf("BuildDeployment() replicas = %v, want %v", *deployment.Spec.Replicas, tt.expectedReplicas)
			}

			container := deployment.Spec.Template.Spec.Containers[0]
			if container.Image != tt.expectedImage {
				t.Errorf("BuildDeployment() image = %v, want %v", container.Image, tt.expectedImage)
			}
		})
	}
}
