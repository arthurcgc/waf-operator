package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	extensionsv1 "github.com/arthurcgc/waf-operator/api/v1"
	"github.com/sirupsen/logrus"
	"github.com/tsuru/nginx-operator/api/v1alpha1"
	nginxv1alpha1 "github.com/tsuru/nginx-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func (r *WafReconciler) getWafInstance(ctx context.Context, objKey types.NamespacedName) (*extensionsv1.Waf, error) {
	var instance extensionsv1.Waf
	if err := r.Client.Get(ctx, objKey, &instance); err != nil {
		return nil, err
	}

	return &instance, nil
}

func (r *WafReconciler) getPlan(ctx context.Context, instance *extensionsv1.Waf) (*extensionsv1.WafPlan, error) {
	planName := types.NamespacedName{
		Name:      instance.Spec.WafPlanName,
		Namespace: instance.Namespace,
	}
	plan := &extensionsv1.WafPlan{}
	err := r.Client.Get(ctx, planName, plan)
	if err != nil {
		return nil, err
	}

	return plan, nil
}

func (r *WafReconciler) renderTemplate(ctx context.Context, instance *extensionsv1.Waf, plan *extensionsv1.WafPlan) (string, error) {
	switch plan.Name {
	case "default":
		return fmt.Sprintf(`
		load_module modules/ngx_http_modsecurity_module.so;
		events {}
	
		http {
			server {
			listen 8080;
	
			modsecurity on;
			modsecurity_rules_file /etc/nginx/extra_files/modsec-includes.conf;
	
			location / {
				proxy_pass %s;
			}
	
			location /nginx-health {
				access_log off;
				return 200 "healthy\n";
			}
		}
		}`, instance.Spec.Bind.Hostname), nil
	}

	return "", errors.New(fmt.Sprintf("plan not found on instance namespace: %s\n can't create nginx.conf", instance.Namespace))
}

func newMainCM(instance *extensionsv1.Waf, renderedTemplate string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-config", instance.Name),
			Namespace: instance.Namespace,
		},

		Data: map[string]string{
			"nginx.conf": renderedTemplate,
		},
	}
}

func newWafConfig(instance *extensionsv1.Waf) *corev1.ConfigMap {
	wafCM := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-conf-extra", instance.Name),
			Namespace: instance.Namespace,
		},

		Data: map[string]string{
			"modsec-includes.conf": `
Include /usr/local/waf-conf/modsecurity-recommended.conf
Include /usr/local/waf-conf/crs-setup.conf
Include /usr/local/waf-conf/rules/*.conf
			`,
		},
	}

	instance.Spec.ExtraFiles = &nginxv1alpha1.FilesRef{
		Name: wafCM.Name,
		Files: map[string]string{
			"modsec-includes.conf": "modsec-includes.conf",
		},
	}

	return wafCM
}

func (r *WafReconciler) reconcileConfigMap(ctx context.Context, configMap *corev1.ConfigMap) error {
	found := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: configMap.ObjectMeta.Name, Namespace: configMap.ObjectMeta.Namespace}, found)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			logrus.Errorf("Failed to get configMap: %v", err)
			return err
		}
		err = r.Client.Create(ctx, configMap)
		if err != nil {
			logrus.Errorf("Failed to create configMap: %v", err)
			return err
		}
		return nil
	}

	configMap.ObjectMeta.ResourceVersion = found.ObjectMeta.ResourceVersion
	err = r.Client.Update(ctx, configMap)
	if err != nil {
		logrus.Errorf("Failed to update configMap: %v", err)
	}
	return err
}

func (r *WafReconciler) getNginx(ctx context.Context, instance *extensionsv1.Waf) (*nginxv1alpha1.Nginx, error) {
	found := &nginxv1alpha1.Nginx{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, found)
	if k8sErrors.IsNotFound(err) {
		return nil, err
	}
	return found, err
}

func (r *WafReconciler) reconcileNginx(ctx context.Context, instance *extensionsv1.Waf, nginx *nginxv1alpha1.Nginx) error {
	found, err := r.getNginx(ctx, instance)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			logrus.Errorf("Failed to get nginx CR: %v", err)
			return err
		}
		err = r.Client.Create(ctx, nginx)
		if err != nil {
			logrus.Errorf("Failed to create nginx CR: %v", err)
			return err
		}
		return nil
	}

	// Update only replicas if rollout is not enabled to ensure HPAs work
	// correctly.
	// if !r.rolloutEnabled(instance) {
	// 	nginx = found
	// 	nginx.Spec.Replicas = instance.Spec.Replicas
	// }

	nginx = found
	nginx.ObjectMeta.ResourceVersion = found.ObjectMeta.ResourceVersion
	err = r.Client.Update(ctx, nginx)
	if err != nil {
		logrus.Errorf("Failed to update nginx CR: %v", err)
	}
	return err
}

func labelsForWafInstance(instance *extensionsv1.Waf) map[string]string {
	return map[string]string{
		"waf.extensions/instance-name": instance.Name,
		"waf.extensions/plan-name":     instance.Spec.WafPlanName,
	}
}

func generateNginxHash(nginx *nginxv1alpha1.Nginx) (string, error) {
	if nginx == nil {
		return "", nil
	}
	nginx = nginx.DeepCopy()
	nginx.Spec.Replicas = nil
	data, err := json.Marshal(nginx.Spec)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(hash[:])), nil
}

func newNginx(instance *extensionsv1.Waf, plan *extensionsv1.WafPlan, mainCM *corev1.ConfigMap) *nginxv1alpha1.Nginx {
	return &nginxv1alpha1.Nginx{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, schema.GroupVersionKind{
					Group:   v1alpha1.GroupVersion.Group,
					Version: v1alpha1.GroupVersion.Version,
					Kind:    "RpaasInstance",
				}),
			},
			Labels: labelsForWafInstance(instance),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Nginx",
			APIVersion: "nginx.tsuru.io/v1alpha1",
		},
		Spec: nginxv1alpha1.NginxSpec{
			Image:    plan.Spec.Image,
			Replicas: instance.Spec.Replicas,
			Config: &nginxv1alpha1.ConfigRef{
				Name: mainCM.Name,
				Kind: nginxv1alpha1.ConfigKindConfigMap,
			},
			Resources:       plan.Spec.Resources,
			Service:         instance.Spec.Service,
			HealthcheckPath: "/nginx-health",
			ExtraFiles:      instance.Spec.ExtraFiles,
		},
	}
}
