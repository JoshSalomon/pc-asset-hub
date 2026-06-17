package controllers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/project-catalyst/pc-asset-hub/internal/operator/api/v1alpha1"
)

type ExporterPluginReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	HTTPClient    *http.Client
	ProbeInterval time.Duration
	TokenGetter   func() string
}

func (r *ExporterPluginReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var cr v1alpha1.ExporterPlugin
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if cr.Status.LastHealthCheck != "" {
		lastCheck, err := time.Parse(time.RFC3339, cr.Status.LastHealthCheck)
		if err == nil {
			staleThreshold := 3 * r.ProbeInterval
			if time.Since(lastCheck) > staleThreshold && cr.Status.Phase != "Unknown" {
				log.Info("Health probe stale, transitioning to Unknown",
					"exporterPlugin", cr.Name,
					"lastCheck", cr.Status.LastHealthCheck)
				cr.Status.Phase = "Unknown"
				cr.Status.Message = fmt.Sprintf("health probe stale — no result for >%s", staleThreshold)
				if err := r.Status().Update(ctx, &cr); err != nil {
					log.Error(err, "failed to update stale status", "name", cr.Name)
					return ctrl.Result{RequeueAfter: r.ProbeInterval}, nil
				}
				return ctrl.Result{RequeueAfter: r.ProbeInterval}, nil
			}
		}
	}

	endpoint := cr.Spec.Endpoint + "/health"
	probeReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ctrl.Result{RequeueAfter: r.ProbeInterval}, nil
	}

	if token := r.TokenGetter(); token != "" {
		probeReq.Header.Set("Authorization", "Bearer "+token)
	}

	var statusCode int
	var probeErr error
	resp, err := r.HTTPClient.Do(probeReq)
	if err != nil {
		probeErr = err
	} else {
		statusCode = resp.StatusCode
		resp.Body.Close()
	}

	phase, message := classifyHealthResult(statusCode, probeErr)

	cr.Status.Phase = phase
	cr.Status.Message = message
	cr.Status.LastHealthCheck = time.Now().UTC().Format(time.RFC3339)

	if err := r.Status().Update(ctx, &cr); err != nil {
		log.Error(err, "failed to update ExporterPlugin status", "name", cr.Name)
		return ctrl.Result{RequeueAfter: r.ProbeInterval}, nil
	}

	return ctrl.Result{RequeueAfter: r.ProbeInterval}, nil
}

func classifyHealthResult(statusCode int, err error) (phase, message string) {
	if err != nil {
		if err == context.DeadlineExceeded {
			return "Unhealthy", "health probe timed out"
		}
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			return "Unhealthy", "health probe timed out"
		}
		return "Error", fmt.Sprintf("plugin unreachable: %s", err)
	}

	if statusCode == http.StatusOK {
		return "Ready", ""
	}

	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return "Error", "health probe authentication rejected"
	}

	return "Unhealthy", fmt.Sprintf("health probe returned HTTP %d", statusCode)
}

func (r *ExporterPluginReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ExporterPlugin{}).
		Named("exporterplugin").
		Complete(r)
}

