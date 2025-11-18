package tunnelclass

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	portalv1alpha1 "github.com/gosuda/portal-expose/api/v1alpha1"
)

const (
	DefaultClassAnnotation = "portal.gosuda.org/is-default-class"
)

// GetTunnelClass returns the TunnelClass to use for a PortalExpose
// It follows this priority:
// 1. Explicit spec.tunnelClassName if set
// 2. Default TunnelClass (annotated with portal.gosuda.org/is-default-class: "true")
// 3. Error if no default exists
func GetTunnelClass(ctx context.Context, c client.Client, tunnelClassName string) (*portalv1alpha1.TunnelClass, error) {
	// If explicit TunnelClass name provided, fetch it
	if tunnelClassName != "" {
		tunnelClass := &portalv1alpha1.TunnelClass{}
		key := client.ObjectKey{Name: tunnelClassName}
		if err := c.Get(ctx, key, tunnelClass); err != nil {
			return nil, fmt.Errorf("failed to get TunnelClass %q: %w", tunnelClassName, err)
		}
		return tunnelClass, nil
	}

	// Otherwise, find the default TunnelClass
	return GetDefaultTunnelClass(ctx, c)
}

// GetDefaultTunnelClass finds the TunnelClass marked as default
func GetDefaultTunnelClass(ctx context.Context, c client.Client) (*portalv1alpha1.TunnelClass, error) {
	tunnelClasses := &portalv1alpha1.TunnelClassList{}
	if err := c.List(ctx, tunnelClasses); err != nil {
		return nil, fmt.Errorf("failed to list TunnelClasses: %w", err)
	}

	// Find the default
	for i := range tunnelClasses.Items {
		tc := &tunnelClasses.Items[i]
		if tc.Annotations != nil && tc.Annotations[DefaultClassAnnotation] == "true" {
			return tc, nil
		}
	}

	return nil, fmt.Errorf("no default TunnelClass found (annotate one with %s: \"true\")", DefaultClassAnnotation)
}
