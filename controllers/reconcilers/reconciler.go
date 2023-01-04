package reconcilers

import (
	"context"
	v1 "github.com/laurafitzgerald/observability-operator/api/v4/v1"
)

type ObservabilityReconciler interface {
	Reconcile(ctx context.Context, cr *v1.Observability, status *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error)
	Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error)
}
