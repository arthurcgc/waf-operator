/*


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

package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	nginxv1alpha1 "github.com/tsuru/nginx-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	wafv1 "github.com/arthurcgc/waf-operator/api/v1"
)

// WafReconciler reconciles a Waf object
type WafReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=waf.arthurcgc.waf-operator,resources=wafs;wafplans,verbs=*
// +kubebuilder:rbac:groups=waf.arthurcgc.waf-operator,resources=wafs/status;wafplans/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services;configmaps;secrets;events;persistentvolumeclaims;endpoints;pods,verbs=*
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;create
// +kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs=*
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;create
// +kubebuilder:rbac:groups=nginx.tsuru.io,resources=*,verbs=*

func (r *WafReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("waf", req.NamespacedName)

	instance, err := r.getWafInstance(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}

	plan, err := r.getPlan(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	rendered, err := r.renderTemplate(ctx, instance, plan)
	if err != nil {
		return reconcile.Result{}, err
	}

	wafCM, err := newWafConfig(instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	err = r.reconcileConfigMap(ctx, wafCM)
	if err != nil {
		return reconcile.Result{}, err
	}

	mainCM := newMainCM(instance, rendered)
	err = r.reconcileConfigMap(ctx, mainCM)
	if err != nil {
		return reconcile.Result{}, err
	}

	nginx := newNginx(instance, plan, mainCM)
	if err = r.reconcileNginx(ctx, instance, nginx); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.refreshStatus(ctx, instance, nginx); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *WafReconciler) refreshStatus(ctx context.Context, instance *wafv1.Waf, newNginx *nginxv1alpha1.Nginx) error {
	existingNginx, err := r.getNginx(ctx, instance)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return err
	}
	newHash, err := generateNginxHash(newNginx)
	if err != nil {
		return err
	}

	existingHash, err := generateNginxHash(existingNginx)
	if err != nil {
		return err
	}

	newStatus := wafv1.WafStatus{
		ObservedGeneration:        instance.Generation,
		WantedNginxRevisionHash:   newHash,
		ObservedNginxRevisionHash: existingHash,
		NginxUpdated:              newHash == existingHash,
	}

	if existingNginx != nil {
		newStatus.CurrentReplicas = existingNginx.Status.CurrentReplicas
		newStatus.PodSelector = existingNginx.Status.PodSelector
	}

	if reflect.DeepEqual(instance.Status, newStatus) {
		return nil
	}

	instance.Status = newStatus
	err = r.Client.Status().Update(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to update rpaas instance status: %v", err)
	}

	return nil
}

func (r *WafReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wafv1.Waf{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&nginxv1alpha1.Nginx{}).
		Complete(r)
}
