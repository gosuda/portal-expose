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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// FinalizerName is the finalizer added to PortalExpose resources
	FinalizerName = "portal.gosuda.org/cleanup-tunnel-deployment"
)

// AddFinalizer adds a finalizer to the object if it doesn't already exist
func AddFinalizer(obj client.Object, finalizer string) bool {
	if !controllerutil.ContainsFinalizer(obj, finalizer) {
		controllerutil.AddFinalizer(obj, finalizer)
		return true
	}
	return false
}

// RemoveFinalizer removes a finalizer from the object if it exists
func RemoveFinalizer(obj client.Object, finalizer string) bool {
	if controllerutil.ContainsFinalizer(obj, finalizer) {
		controllerutil.RemoveFinalizer(obj, finalizer)
		return true
	}
	return false
}

// HasFinalizer checks if the object has the specified finalizer
func HasFinalizer(obj client.Object, finalizer string) bool {
	return controllerutil.ContainsFinalizer(obj, finalizer)
}
