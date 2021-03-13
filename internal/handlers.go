package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (w *WafManager) deployNginx(ctx context.Context, opts DeployOpts, configMapName string) error {
	nginxGVR := schema.GroupVersionResource{Group: "nginx.tsuru.io", Version: "v1alpha1", Resource: "nginxes"}

	nginx := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "nginx.tsuru.io/v1alpha1",
			"kind":       "Nginx",
			"metadata": map[string]interface{}{
				"name": opts.Name,
			},
			"spec": map[string]interface{}{
				"replicas": opts.Replicas,
				"image":    wafImage,
				"config": map[string]interface{}{
					"kind": "ConfigMap",
					"name": configMapName,
				},
				"service": map[string]interface{}{
					"type": "NodePort",
				},
			},
		},
	}

	_, err := w.dynamicClient.Resource(nginxGVR).Namespace(opts.Namespace).Create(ctx, nginx, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (w *WafManager) deployConf(ctx context.Context, opts DeployOpts, configMapName string) error {
	immutable := new(bool)
	configMap := &v1.ConfigMap{
		metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: opts.Namespace,
		},

		immutable,
		map[string]string{
			"nginx.conf": `
	events {}

	http {
		server {
		listen 8080;

		location / {
			proxy_pass http://go-hostname.backend.svc.cluster.local:80;
		}
	}
	}`,
		},
		nil,
	}
	_, err := w.defaultClient.CoreV1().ConfigMaps(opts.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (w *WafManager) deploy(c echo.Context) error {
	ctx := c.Request().Context()
	var opts DeployOpts
	err := json.NewDecoder(c.Request().Body).Decode(&opts)
	if err != nil {
		return err
	}

	configMapName := fmt.Sprintf("%s-conf", opts.Name)
	if err := w.deployConf(ctx, opts, configMapName); err != nil {
		return err
	}
	if err := w.deployNginx(ctx, opts, configMapName); err != nil {
		return err
	}

	return c.String(http.StatusCreated, "Created nginx resource!")
}

func healthcheck(c echo.Context) error {
	return c.String(http.StatusOK, "WORKING")
}
