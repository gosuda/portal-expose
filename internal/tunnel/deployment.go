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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	portalv1alpha1 "github.com/gosuda/portal-expose/api/v1alpha1"
)

const (
	// TunnelImage is the default tunnel container image
	TunnelImage = "ghcr.io/gosuda/portal-tunnel:1.0.0"
)

// BuildDeployment creates a Deployment spec for tunnel pods
func BuildDeployment(portalExpose *portalv1alpha1.PortalExpose, tunnelClass *portalv1alpha1.TunnelClass) *appsv1.Deployment {
	name := portalExpose.Name + "-tunnel"
	namespace := portalExpose.Namespace

	labels := map[string]string{
		"app.kubernetes.io/name":         "portal-tunnel",
		"app.kubernetes.io/component":    "tunnel",
		"app.kubernetes.io/managed-by":   "portal-expose-controller",
		"portal.gosuda.org/portalexpose": portalExpose.Name,
	}

	// Container args matching portal-tunnel command:
	// bin/portal-tunnel expose --relay <url> [--relay <url> ...] --host localhost --port 8080 --name <service>
	args := []string{
		"expose",
		"--name", portalExpose.Spec.App.Name,
		"--host", fmt.Sprintf("%s.%s.svc.cluster.local", portalExpose.Spec.App.Service.Name, portalExpose.Namespace),
		"--port", fmt.Sprintf("%d", portalExpose.Spec.App.Service.Port),
	}
	// Add all relay URLs
	for _, target := range portalExpose.Spec.Relay.Targets {
		args = append(args, "--relay", target.URL)
	}

	// Get resources for size
	resources := GetResourcesForSize(tunnelClass.Spec.Size)

	// Create deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &tunnelClass.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       func() *intstr.IntOrString { v := intstr.FromString("25%"); return &v }(),
					MaxUnavailable: func() *intstr.IntOrString { v := intstr.FromString("25%"); return &v }(),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      "tunnel",
							Image:     TunnelImage,
							Args:      args,
							Resources: resources,
						},
					},
					NodeSelector: tunnelClass.Spec.NodeSelector,
					Tolerations:  tunnelClass.Spec.Tolerations,
				},
			},
		},
	}

	return deployment
}

// GetResourcesForSize returns resource requirements for a given size
func GetResourcesForSize(size string) corev1.ResourceRequirements {
	switch size {
	case "small":
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
		}
	case "medium":
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("250m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		}
	case "large":
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2000m"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
		}
	default:
		// Default to small if size is unrecognized
		return GetResourcesForSize("small")
	}
}
