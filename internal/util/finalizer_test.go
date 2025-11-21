package util

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// MockObject is a simple struct that implements client.Object for testing
type MockObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func (m *MockObject) DeepCopyObject() runtime.Object {
	return &MockObject{
		TypeMeta:   m.TypeMeta,
		ObjectMeta: *m.DeepCopy(),
	}
}

func TestRemoveFinalizer(t *testing.T) {
	finalizer := "example.com/finalizer"

	tests := []struct {
		name           string
		initial        []string
		wantRemoved    bool
		wantFinalizers []string
	}{
		{
			name:           "Remove existing finalizer",
			initial:        []string{finalizer, "other-finalizer"},
			wantRemoved:    true,
			wantFinalizers: []string{"other-finalizer"},
		},
		{
			name:           "Remove non-existent finalizer",
			initial:        []string{"other-finalizer"},
			wantRemoved:    false,
			wantFinalizers: []string{"other-finalizer"},
		},
		{
			name:           "Remove from empty list",
			initial:        []string{},
			wantRemoved:    false,
			wantFinalizers: []string(nil), // controllerutil might return nil or empty slice
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &MockObject{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: tt.initial,
				},
			}

			removed := RemoveFinalizer(obj, finalizer)

			if removed != tt.wantRemoved {
				t.Errorf("RemoveFinalizer() = %v, want %v", removed, tt.wantRemoved)
			}

			// Verify finalizers list
			if len(obj.GetFinalizers()) != len(tt.wantFinalizers) {
				t.Errorf("Finalizers = %v, want %v", obj.GetFinalizers(), tt.wantFinalizers)
			}

			// Check if finalizer is gone
			if controllerutil.ContainsFinalizer(obj, finalizer) {
				t.Errorf("Finalizer %s was not removed", finalizer)
			}
		})
	}
}

func TestAddFinalizer(t *testing.T) {
	finalizer := "example.com/finalizer"

	tests := []struct {
		name           string
		initial        []string
		wantAdded      bool
		wantFinalizers []string
	}{
		{
			name:           "Add new finalizer",
			initial:        []string{"other-finalizer"},
			wantAdded:      true,
			wantFinalizers: []string{"other-finalizer", finalizer},
		},
		{
			name:           "Add existing finalizer",
			initial:        []string{finalizer},
			wantAdded:      false,
			wantFinalizers: []string{finalizer},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &MockObject{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: tt.initial,
				},
			}

			added := AddFinalizer(obj, finalizer)

			if added != tt.wantAdded {
				t.Errorf("AddFinalizer() = %v, want %v", added, tt.wantAdded)
			}

			if !controllerutil.ContainsFinalizer(obj, finalizer) {
				t.Errorf("Finalizer %s was not added", finalizer)
			}
		})
	}
}
