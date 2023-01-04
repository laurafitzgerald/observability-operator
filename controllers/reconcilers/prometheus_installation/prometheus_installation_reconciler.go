package prometheus_installation

import (
	"context"

	"github.com/go-logr/logr"
	coreosv1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "github.com/laurafitzgerald/observability-operator/api/v4/v1"
	"github.com/redhat-developer/observability-operator/v4/controllers/model"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers"
	"github.com/redhat-developer/observability-operator/v4/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler struct {
	client client.Client
	logger logr.Logger
	scheme *runtime.Scheme
}

func NewReconciler(client client.Client, logger logr.Logger, scheme *runtime.Scheme) reconcilers.ObservabilityReconciler {
	return &Reconciler{
		client: client,
		logger: logger,
		scheme: scheme,
	}
}

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	// Delete subscription
	subscription := model.GetPrometheusSubscription(cr)
	err := r.client.Delete(ctx, subscription)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// Delete operatorgroup
	operatorgroup := model.GetPrometheusOperatorgroup(cr)
	err = r.client.Delete(ctx, operatorgroup)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	// Delete catalog source
	source := model.GetPrometheusCatalogSource(cr)
	err = r.client.Delete(ctx, source)
	if err != nil && !errors.IsNotFound(err) {
		return v1.ResultFailed, err
	}

	if cr.DescopedModeEnabled() {
		namespace := model.GetPrometheusNamespace(cr)
		err = r.client.Delete(ctx, namespace)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	if cr.DescopedModeEnabled() {
		status, err := r.reconcileNamespace(ctx, cr)
		if err != nil {
			return status, err
		}
	}

	// Catalog source
	status, err := r.reconcileCatalogSource(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// Prometheus subscription
	status, err = r.reconcileSubscription(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	// Observability operator group
	status, err = r.reconcileOperatorgroup(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	status, err = r.waitForPrometheusOperator(ctx, cr)
	if status != v1.ResultSuccess {
		return status, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) waitForPrometheusOperator(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	deployments := &appsv1.DeploymentList{}
	opts := &client.ListOptions{
		Namespace: cr.GetPrometheusOperatorNamespace(),
	}
	err := r.client.List(ctx, deployments, opts)
	if err != nil {
		return v1.ResultFailed, err
	}

	for _, deployment := range deployments.Items {
		if deployment.Name == "prometheus-operator" {
			if deployment.Status.ReadyReplicas > 0 {
				return v1.ResultSuccess, nil
			}
		}
	}
	return v1.ResultInProgress, nil
}

func (r *Reconciler) reconcileNamespace(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	namespace := model.GetPrometheusNamespace(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, namespace, func() error {
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileCatalogSource(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	source := &v1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-catalogsource",
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}
	selector := client.ObjectKey{
		Namespace: source.Namespace,
		Name:      source.Name,
	}

	// look for catalogSource for Prometheus Operator index. If not found check for unwanted Operator and remove
	err := r.client.Get(ctx, selector, source)
	if err != nil {
		if !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}
		err = r.removePrometheusOperatorResources(ctx, cr)
		if err != nil {
			return v1.ResultFailed, err
		}
	}

	// install Promethues Operator by catalogSource
	_, err = controllerutil.CreateOrUpdate(ctx, r.client, source, func() error {
		source.Spec = v1alpha1.CatalogSourceSpec{
			SourceType: v1alpha1.SourceTypeGrpc,
			Image:      "quay.io/integreatly/custom-prometheus-index:1.0.0",
		}
		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	subscription := model.GetPrometheusSubscription(cr)
	source := model.GetPrometheusCatalogSource(cr)

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, subscription, func() error {
		subscription.Spec = &v1alpha1.SubscriptionSpec{
			CatalogSource:          source.Name,
			CatalogSourceNamespace: cr.GetPrometheusOperatorNamespace(),
			Package:                "prometheus",
			Channel:                "preview",
			InstallPlanApproval:    v1alpha1.ApprovalAutomatic,
			Config:                 &v1alpha1.SubscriptionConfig{Resources: model.GetPrometheusOperatorResourceRequirement(cr)},
		}

		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}
func (r *Reconciler) reconcileOperatorgroup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	exists, err := utils.HasOperatorGroupForNamespace(ctx, r.client, cr.Namespace)
	if err != nil {
		return v1.ResultFailed, err
	}

	if exists {
		return v1.ResultSuccess, nil
	}

	operatorgroup := model.GetPrometheusOperatorgroup(cr)

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, operatorgroup, func() error {
		operatorgroup.Spec = coreosv1.OperatorGroupSpec{
			TargetNamespaces: []string{cr.GetPrometheusOperatorNamespace()},
		}

		return nil
	})

	if err != nil {
		return v1.ResultFailed, err
	}

	return v1.ResultSuccess, nil
}

func (r *Reconciler) removePrometheusOperatorResources(ctx context.Context, cr *v1.Observability) error {
	// Delete subscription
	subscription := &v1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-subscription",
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
		Spec: &v1alpha1.SubscriptionSpec{
			StartingCSV: "prometheusoperator.0.56.3",
		},
	}
	err := r.client.Delete(ctx, subscription)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	// Delete csv to uninstall unwanted operator
	csv := &v1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheusoperator.0.56.3",
			Namespace: cr.GetPrometheusOperatorNamespace(),
		},
	}
	err = r.client.Delete(ctx, csv)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}
