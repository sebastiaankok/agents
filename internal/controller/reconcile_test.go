package controller_test

import (
	"context"
	"testing"

	"github.com/sebastiaankok/agents/internal/controller"
)

func TestReconcile_NoOp(t *testing.T) {
	err := controller.Reconcile(context.Background())
	if err != nil {
		t.Errorf("Reconcile() error = %v, want nil", err)
	}
}
