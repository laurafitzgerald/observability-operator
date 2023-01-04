package configuration

import (
	"context"

	"github.com/blang/semver"
	"github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	v1 "github.com/laurafitzgerald/observability-operator/api/v4/v1"
	"github.com/redhat-developer/observability-operator/v4/controllers/model"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	GrafanaBaseImage = "docker.io/grafana/grafana:"
)

func (r *Reconciler) reconcileGrafanaCr(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) error {
	grafana := model.GetGrafanaCr(cr)

	var f = false
	var t = true
	var replicaCount int32 = 1

	specVer, verError := semver.ParseTolerant(model.GetGrafanaVersion(indexes, cr))
	GrafanaImage := ""
	if specVer.String() != "0.0.0" && verError == nil {
		GrafanaImage = GrafanaBaseImage + specVer.String()
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.client, grafana, func() error {
		grafana.Spec = v1alpha1.GrafanaSpec{
			Config: v1alpha1.GrafanaConfig{
				Log: &v1alpha1.GrafanaConfigLog{
					Mode:  "console",
					Level: "warn",
				},
				Auth: &v1alpha1.GrafanaConfigAuth{
					DisableLoginForm:   &f,
					DisableSignoutMenu: &t,
				},
				AuthBasic: &v1alpha1.GrafanaConfigAuthBasic{
					Enabled: &t,
				},
				AuthAnonymous: &v1alpha1.GrafanaConfigAuthAnonymous{
					Enabled: &t,
				},
			},
			Containers: []core.Container{
				{
					Name:  "grafana-proxy",
					Image: "quay.io/openshift/origin-oauth-proxy:4.8",
					Args: []string{
						"-provider=openshift",
						"-pass-basic-auth=false",
						"-https-address=:9091",
						"-http-address=",
						"-email-domain=*",
						"-upstream=http://localhost:3000",
						"-openshift-sar={\"resource\": \"namespaces\", \"verb\": \"get\"}",
						"-openshift-delegate-urls={\"/\": {\"resource\": \"namespaces\", \"verb\": \"get\"}}",
						"-tls-cert=/etc/tls/private/tls.crt",
						"-tls-key=/etc/tls/private/tls.key",
						"-client-secret-file=/var/run/secrets/kubernetes.io/serviceaccount/token",
						"-cookie-secret-file=/etc/proxy/secrets/session_secret",
						"-openshift-service-account=grafana-serviceaccount",
						"-openshift-ca=/etc/pki/tls/cert.pem",
						"-openshift-ca=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
						"-skip-auth-regex=^/metrics",
					},
					Ports: []core.ContainerPort{
						{
							Name:          "grafana-proxy",
							ContainerPort: 9091,
						},
					},
					Resources: core.ResourceRequirements{},
					VolumeMounts: []core.VolumeMount{
						{
							Name:      "secret-grafana-k8s-tls",
							ReadOnly:  false,
							MountPath: "/etc/tls/private",
						},
						{
							Name:      "secret-grafana-k8s-proxy",
							ReadOnly:  false,
							MountPath: "/etc/proxy/secrets",
						},
					},
				},
			},
			BaseImage: GrafanaImage,
			DashboardLabelSelector: []*metav1.LabelSelector{
				model.GetGrafanaDashboardLabelSelectors(cr, indexes),
			},
			Ingress: &v1alpha1.GrafanaIngress{
				Enabled:     true,
				TargetPort:  "grafana-proxy",
				Termination: "reencrypt",
			},
			Secrets: []string{
				"grafana-k8s-tls",
				"grafana-k8s-proxy",
			},
			Service: &v1alpha1.GrafanaService{
				Annotations: map[string]string{
					"service.alpha.openshift.io/serving-cert-secret-name": "grafana-k8s-tls",
				},
				Ports: []core.ServicePort{
					{
						Name:       "grafana-proxy",
						Protocol:   "TCP",
						Port:       9091,
						TargetPort: intstr.FromString("grafana-proxy"),
					},
				},
			},
			ServiceAccount: &v1alpha1.GrafanaServiceAccount{
				Annotations: map[string]string{
					"serviceaccounts.openshift.io/oauth-redirectreference.primary": "{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"grafana-route\"}}",
				},
			},
			Client: &v1alpha1.GrafanaClient{
				PreferService: &t,
			},
			Deployment: &v1alpha1.GrafanaDeployment{
				Replicas:          &replicaCount,
				PriorityClassName: model.ObservabilityPriorityClassName,
				Annotations: map[string]string{
					"cluster-autoscaler.kubernetes.io/safe-to-evict": "true",
				},
			},
			Resources: model.GetGrafanaResourceRequirement(cr),
		}
		if cr.Spec.Tolerations != nil {
			grafana.Spec.Deployment.Tolerations = cr.Spec.Tolerations
		}
		if cr.Spec.Affinity != nil {
			grafana.Spec.Deployment.Affinity = cr.Spec.Affinity
		}
		return nil
	})

	return err
}
