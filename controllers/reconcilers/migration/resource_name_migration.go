package migration

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v1 "github.com/laurafitzgerald/observability-operator/api/v4/v1"
	"github.com/redhat-developer/observability-operator/v4/controllers/model"
	"github.com/redhat-developer/observability-operator/v4/controllers/reconcilers"
	"github.com/redhat-developer/observability-operator/v4/controllers/utils"
	"k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client client.Client
	logger logr.Logger
}

func NewReconciler(client client.Client, logger logr.Logger) reconcilers.ObservabilityReconciler {
	return &Reconciler{
		client: client,
		logger: logger,
	}
}

func (r *Reconciler) Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error) {
	return v1.ResultSuccess, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.Observability, s *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error) {
	if s.Migrated {
		return v1.ResultSuccess, nil
	}

	//check if Prometheus resources needed to be migrated
	if cr.Spec.PrometheusDefaultName == "" {
		//remove Prometheus resources
		prometheusCr := model.GetPrometheus(cr)
		prometheusCr.Name = model.PrometheusOldDefaultName

		err := r.client.Delete(ctx, prometheusCr)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		utils.WaitForPrometheusToBeRemoved(ctx, cr, r.client)

		prometheusRoute := model.GetPrometheusRoute(cr)
		prometheusRoute.Name = model.PrometheusOldDefaultName
		err = r.client.Delete(ctx, prometheusRoute)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		prometheusService := model.GetPrometheusService(cr)
		prometheusService.Name = model.PrometheusOldDefaultName
		err = r.client.Delete(ctx, prometheusService)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		prometheusServiceAccount := model.GetPrometheusServiceAccount(cr)
		prometheusServiceAccount.Name = model.PrometheusOldDefaultName
		err = r.client.Delete(ctx, prometheusServiceAccount)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		prometheusClusterRole := model.GetPrometheusClusterRole(cr)
		prometheusClusterRole.Name = model.PrometheusOldDefaultName
		err = r.client.Delete(ctx, prometheusClusterRole)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		prometheusClusterRoleBinding := model.GetPrometheusClusterRoleBinding(cr)
		prometheusClusterRoleBinding.Name = model.PrometheusOldDefaultName
		err = r.client.Delete(ctx, prometheusClusterRoleBinding)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}
	}

	//check if Alertmanager resources need migration
	if cr.Spec.AlertManagerDefaultName == "" {
		//remove Alertmanager resources
		overrideSecret, _ := cr.HasAlertmanagerConfigSecret()
		if !overrideSecret && !cr.ExternalSyncDisabled() {
			alertmanagerSecret := model.GetAlertmanagerSecret(cr)
			alertmanagerSecret.Name = fmt.Sprintf("alertmanager-%s", model.AlertManagerOldDefaultName)
			err := r.client.Delete(ctx, alertmanagerSecret)
			if err != nil && !errors.IsNotFound(err) {
				return v1.ResultFailed, err
			}
		}

		alertManagerCr := model.GetAlertmanagerCr(cr)
		alertManagerCr.Name = model.AlertManagerOldDefaultName
		err := r.client.Delete(ctx, alertManagerCr)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		utils.WaitForAlertmanagerToBeRemoved(ctx, cr, r.client)

		alertmanagerServiceAccount := model.GetAlertmanagerServiceAccount(cr)
		alertmanagerServiceAccount.Name = model.AlertManagerOldDefaultName
		err = r.client.Delete(ctx, alertmanagerServiceAccount)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		alertmanagerRoute := model.GetAlertmanagerRoute(cr)
		alertmanagerRoute.Name = model.AlertManagerOldDefaultName
		err = r.client.Delete(ctx, alertmanagerRoute)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		alertmanagerService := model.GetAlertmanagerService(cr)
		alertmanagerService.Name = model.AlertManagerOldDefaultName
		err = r.client.Delete(ctx, alertmanagerService)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		alertmanagerClusterRole := model.GetAlertmanagerClusterRole(cr)
		alertmanagerClusterRole.Name = model.AlertManagerOldDefaultName
		err = r.client.Delete(ctx, alertmanagerClusterRole)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		alertmanagerClusterRoleBinding := model.GetAlertmanagerClusterRoleBinding(cr)
		alertmanagerClusterRoleBinding.Name = model.AlertManagerOldDefaultName
		err = r.client.Delete(ctx, alertmanagerClusterRoleBinding)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}
	}

	//check if Grafana CR need migration
	if cr.Spec.GrafanaDefaultName == "" {
		//remove Grafana resources
		grafanaCR := model.GetGrafanaCr(cr)
		grafanaCR.Name = model.GrafanaOldDefaultName
		err := r.client.Delete(ctx, grafanaCR)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		utils.WaitForGrafanaToBeRemoved(ctx, cr, r.client)
	}

	//check if Promtail resources need migration
	if !cr.ExternalSyncDisabled() || !cr.ObservatoriumDisabled() {
		//remove Promtail rsources
		promtailServiceAccount := model.GetPromtailServiceAccount(cr)
		promtailServiceAccount.Name = model.PromtailOldDefaultName
		err := r.client.Delete(ctx, promtailServiceAccount)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		promtailClusterRole := model.GetPromtailClusterRole(cr)
		promtailClusterRole.Name = model.PromtailOldDefaultName
		err = r.client.Delete(ctx, promtailClusterRole)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}

		promtailClusterRoleBinding := model.GetPromtailClusterRoleBinding(cr)
		promtailClusterRoleBinding.Name = model.PromtailOldDefaultName
		err = r.client.Delete(ctx, promtailClusterRoleBinding)
		if err != nil && !errors.IsNotFound(err) {
			return v1.ResultFailed, err
		}
	}

	s.Migrated = true

	return v1.ResultSuccess, nil
}
