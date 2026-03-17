package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	v1alpha1 "github.com/project-catalyst/pc-asset-hub/internal/operator/api/v1alpha1"
)

// AssetHubReconciler reconciles AssetHub objects.
type AssetHubReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile handles a single reconciliation loop iteration.
func (r *AssetHubReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var cr v1alpha1.AssetHub
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if errors.IsNotFound(err) {
			log.Info("AssetHub resource not found, likely deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// If the CR is being deleted, clean up owned resources
	if !cr.DeletionTimestamp.IsZero() {
		log.Info("AssetHub is being deleted, cleanup handled by ownership")
		return ctrl.Result{}, nil
	}

	// Use the pure reconciler to determine desired state
	spec := AssetHubSpec{
		Replicas:     cr.Spec.Replicas,
		DBConnection: cr.Spec.DBConnection,
		UIReplicas:   cr.Spec.UIReplicas,
		Environment:  cr.Spec.Environment,
		APINodePort:  cr.Spec.APINodePort,
		UINodePort:   cr.Spec.UINodePort,
		APIHostname:  cr.Spec.APIHostname,
		UIHostname:   cr.Spec.UIHostname,
		CORSOrigins:  cr.Spec.CORSOrigins,
		LogLevel:     cr.Spec.LogLevel,
		ClusterRole:  cr.Spec.ClusterRole,
	}
	desired := ReconcileAssetHub(spec)

	// Create or update ConfigMaps
	for _, cms := range desired.ConfigMaps {
		if err := r.reconcileConfigMap(ctx, &cr, cms); err != nil {
			return ctrl.Result{}, r.updateStatus(ctx, &cr, false, fmt.Sprintf("failed to reconcile configmap %s: %v", cms.Name, err))
		}
	}

	// Create or update Deployments
	for _, ds := range desired.Deployments {
		if err := r.reconcileDeployment(ctx, &cr, ds); err != nil {
			return ctrl.Result{}, r.updateStatus(ctx, &cr, false, fmt.Sprintf("failed to reconcile deployment %s: %v", ds.Name, err))
		}
	}

	// Create or update Services
	for _, ss := range desired.Services {
		if err := r.reconcileService(ctx, &cr, ss); err != nil {
			return ctrl.Result{}, r.updateStatus(ctx, &cr, false, fmt.Sprintf("failed to reconcile service %s: %v", ss.Name, err))
		}
	}

	// Create or update Routes (OpenShift only)
	for _, rs := range desired.Routes {
		if err := r.reconcileRoute(ctx, &cr, rs); err != nil {
			return ctrl.Result{}, r.updateStatus(ctx, &cr, false, fmt.Sprintf("failed to reconcile route %s: %v", rs.Name, err))
		}
	}

	// Reconcile CatalogVersion CRs: set owner references and update status
	if err := r.reconcileCatalogVersions(ctx, &cr); err != nil {
		return ctrl.Result{}, r.updateStatus(ctx, &cr, false, fmt.Sprintf("failed to reconcile catalog versions: %v", err))
	}

	// Reconcile Catalog CRs: set owner references, update status, increment DataVersion
	if err := r.reconcileCatalogs(ctx, &cr); err != nil {
		return ctrl.Result{}, r.updateStatus(ctx, &cr, false, fmt.Sprintf("failed to reconcile catalogs: %v", err))
	}

	if err := r.updateStatus(ctx, &cr, true, "all resources reconciled"); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AssetHubReconciler) reconcileConfigMap(ctx context.Context, cr *v1alpha1.AssetHub, cms ConfigMapSpec) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cms.Name,
			Namespace: cr.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Data = cms.Data
		return controllerutil.SetControllerReference(cr, cm, r.Scheme)
	})
	return err
}

func (r *AssetHubReconciler) reconcileDeployment(ctx context.Context, cr *v1alpha1.AssetHub, ds DeploymentSpec) error {
	replicas := int32(ds.Replicas)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ds.Name,
			Namespace: cr.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
		dep.Spec.Replicas = &replicas
		dep.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": ds.Name},
		}

		container := corev1.Container{
			Name:            ds.Name,
			Image:           ds.Image,
			ImagePullPolicy: corev1.PullPolicy(ds.ImagePullPolicy),
			Ports: []corev1.ContainerPort{
				{ContainerPort: ds.ContainerPort},
			},
		}

		// Add envFrom if specified
		if ds.EnvFrom != "" {
			container.EnvFrom = []corev1.EnvFromSource{
				{
					ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: ds.EnvFrom},
					},
				},
			}
			// Inject WATCH_NAMESPACE from the pod's namespace via downward API
			container.Env = []corev1.EnvVar{
				{
					Name: "WATCH_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						},
					},
				},
			}
		}

		// Add readiness probe
		if ds.ReadinessPath != "" {
			container.ReadinessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: ds.ReadinessPath,
						Port: intstr.FromInt32(ds.ContainerPort),
					},
				},
				InitialDelaySeconds: 5,
				PeriodSeconds:       5,
			}
		}

		// Add liveness probe
		if ds.LivenessPath != "" {
			container.LivenessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: ds.LivenessPath,
						Port: intstr.FromInt32(ds.ContainerPort),
					},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       10,
			}
		}

		dep.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": ds.Name},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: ds.ServiceAccountName,
				Containers:         []corev1.Container{container},
			},
		}
		return controllerutil.SetControllerReference(cr, dep, r.Scheme)
	})
	return err
}

func (r *AssetHubReconciler) reconcileService(ctx context.Context, cr *v1alpha1.AssetHub, ss ServiceSpec) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ss.Name,
			Namespace: cr.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Spec.Selector = map[string]string{"app": ss.Name[:len(ss.Name)-4]} // remove "-svc" suffix
		svc.Spec.Type = corev1.ServiceType(ss.Type)

		port := corev1.ServicePort{
			Port:       int32(ss.Port),
			TargetPort: intstr.FromInt32(int32(ss.Port)),
		}
		if ss.NodePort > 0 {
			port.NodePort = ss.NodePort
		}
		svc.Spec.Ports = []corev1.ServicePort{port}

		return controllerutil.SetControllerReference(cr, svc, r.Scheme)
	})
	return err
}

func (r *AssetHubReconciler) reconcileRoute(ctx context.Context, cr *v1alpha1.AssetHub, rs RouteSpec) error {
	route := &unstructured.Unstructured{}
	route.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "route.openshift.io",
		Version: "v1",
		Kind:    "Route",
	})

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(route.GroupVersionKind())
	err := r.Get(ctx, types.NamespacedName{Name: rs.Name, Namespace: cr.Namespace}, existing)

	route.SetName(rs.Name)
	route.SetNamespace(cr.Namespace)
	route.SetLabels(map[string]string{"app.kubernetes.io/managed-by": "assethub-operator"})

	spec := map[string]interface{}{
		"host": rs.Hostname,
		"to": map[string]interface{}{
			"kind": "Service",
			"name": rs.ServiceName,
		},
		"port": map[string]interface{}{
			"targetPort": int64(rs.ServicePort),
		},
	}
	if rs.TLS {
		spec["tls"] = map[string]interface{}{
			"termination": "edge",
		}
	}
	route.Object["spec"] = spec

	// Set owner reference
	ownerRef := metav1.OwnerReference{
		APIVersion: cr.APIVersion,
		Kind:       cr.Kind,
		Name:       cr.Name,
		UID:        cr.UID,
	}
	route.SetOwnerReferences([]metav1.OwnerReference{ownerRef})

	if errors.IsNotFound(err) {
		return r.Create(ctx, route)
	} else if err != nil {
		return err
	}

	// Update existing route
	existing.Object["spec"] = spec
	existing.SetLabels(route.GetLabels())
	existing.SetOwnerReferences(route.GetOwnerReferences())
	return r.Update(ctx, existing)
}

func (r *AssetHubReconciler) updateStatus(ctx context.Context, cr *v1alpha1.AssetHub, ready bool, message string) error {
	latest := &v1alpha1.AssetHub{}
	if err := r.Get(ctx, types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}, latest); err != nil {
		return err
	}
	latest.Status.Ready = ready
	latest.Status.Message = message
	return r.Status().Update(ctx, latest)
}

func (r *AssetHubReconciler) reconcileCatalogVersions(ctx context.Context, cr *v1alpha1.AssetHub) error {
	var cvList v1alpha1.CatalogVersionList
	if err := r.List(ctx, &cvList, client.InNamespace(cr.Namespace)); err != nil {
		return err
	}

	for i := range cvList.Items {
		cv := &cvList.Items[i]

		// Set owner reference if not already set
		if !hasOwnerRef(cv.OwnerReferences, cr.UID) {
			if err := controllerutil.SetOwnerReference(cr, cv, r.Scheme); err != nil {
				return fmt.Errorf("failed to set owner reference on CatalogVersion %s: %w", cv.Name, err)
			}
			if err := r.Update(ctx, cv); err != nil {
				return fmt.Errorf("failed to update CatalogVersion %s: %w", cv.Name, err)
			}
		}

		// Update status using pure reconciler function
		statusResult := ReconcileCatalogVersionStatus(cv.Spec.LifecycleStage)
		if cv.Status.Ready != statusResult.Ready || cv.Status.Message != statusResult.Message {
			cv.Status.Ready = statusResult.Ready
			cv.Status.Message = statusResult.Message
			if err := r.Status().Update(ctx, cv); err != nil {
				return fmt.Errorf("failed to update CatalogVersion %s status: %w", cv.Name, err)
			}
		}
	}

	return nil
}

func (r *AssetHubReconciler) reconcileCatalogs(ctx context.Context, cr *v1alpha1.AssetHub) error {
	var catList v1alpha1.CatalogList
	if err := r.List(ctx, &catList, client.InNamespace(cr.Namespace)); err != nil {
		return err
	}

	for i := range catList.Items {
		cat := &catList.Items[i]

		// Set owner reference if not already set
		if !hasOwnerRef(cat.OwnerReferences, cr.UID) {
			if err := controllerutil.SetOwnerReference(cr, cat, r.Scheme); err != nil {
				return fmt.Errorf("failed to set owner reference on Catalog %s: %w", cat.Name, err)
			}
			if err := r.Update(ctx, cat); err != nil {
				return fmt.Errorf("failed to update Catalog %s: %w", cat.Name, err)
			}
		}

		// Update status and increment DataVersion only when status is stale
		// (not ready, or spec generation changed since last reconcile)
		statusResult := ReconcileCatalogStatus(cat.Generation, cat.Status)
		if statusResult.NeedsUpdate {
			cat.Status.Ready = statusResult.Ready
			cat.Status.Message = statusResult.Message
			cat.Status.DataVersion = statusResult.DataVersion
			cat.Status.ObservedGeneration = statusResult.ObservedGeneration
			if err := r.Status().Update(ctx, cat); err != nil {
				return fmt.Errorf("failed to update Catalog %s status: %w", cat.Name, err)
			}
		}
	}

	return nil
}

func hasOwnerRef(refs []metav1.OwnerReference, uid types.UID) bool {
	for _, ref := range refs {
		if ref.UID == uid {
			return true
		}
	}
	return false
}

// SetupWithManager registers the controller with the manager.
func (r *AssetHubReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// mapToAssetHub maps any CR event to the AssetHub CR in the same namespace.
	// This ensures newly created CRs (without owner refs) trigger reconciliation.
	mapToAssetHub := handler.EnqueueRequestsFromMapFunc(
		func(ctx context.Context, obj client.Object) []ctrl.Request {
			var assetHubList v1alpha1.AssetHubList
			if err := r.List(ctx, &assetHubList, client.InNamespace(obj.GetNamespace())); err != nil {
				return nil
			}
			var requests []ctrl.Request
			for _, ah := range assetHubList.Items {
				requests = append(requests, ctrl.Request{
					NamespacedName: types.NamespacedName{Name: ah.Name, Namespace: ah.Namespace},
				})
			}
			return requests
		},
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.AssetHub{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&v1alpha1.CatalogVersion{}).
		Owns(&v1alpha1.Catalog{}).
		Watches(&v1alpha1.CatalogVersion{}, mapToAssetHub, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&v1alpha1.Catalog{}, mapToAssetHub, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
