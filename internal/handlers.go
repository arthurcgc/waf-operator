package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/tsuru/nginx-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (w *WafManager) deployNginx(ctx context.Context, opts DeployOpts, configMapName, wafConf string) error {
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
				// ExtraFiles references to additional files into a object in the cluster.
				// These additional files will be mounted on `/etc/nginx/extra_files`.
				"extraFiles": &v1alpha1.FilesRef{
					wafConf,
					map[string]string{
						"rules.conf":       "rules.conf",
						"modsecurity.conf": "modsecurity.conf",
					},
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

func (w *WafManager) deployConf(ctx context.Context, opts DeployOpts, configMapName, wafConfName string) error {
	immutable := new(bool)
	wafConf := &v1.ConfigMap{
		metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		metav1.ObjectMeta{
			Name:      wafConfName,
			Namespace: opts.Namespace,
		},

		immutable,
		map[string]string{
			"modsecurity.conf": recomended,
			"rules.conf": `
# Include the recommended configuration
Include /etc/nginx/extra_files/modsecurity.conf
# A test rule
SecRule ARGS:testparam "@contains test" "id:1234,deny,log,status:403"
`,
		},
		nil,
	}
	_, err := w.defaultClient.CoreV1().ConfigMaps(opts.Namespace).Create(ctx, wafConf, metav1.CreateOptions{})
	if err != nil {
		return err
	}
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
	load_module modules/ngx_http_modsecurity_module.so;
	events {}

	http {
		server {
		listen 8080;
		modsecurity on;
		modsecurity_rules_file /etc/nginx/extra_files/rules.conf;

		location / {
			proxy_pass http://go-hostname.backend.svc.cluster.local:80;
		}
	}
	}`,
		},
		nil,
	}
	_, err = w.defaultClient.CoreV1().ConfigMaps(opts.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
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
	wafConf := fmt.Sprintf("%s-conf-extra", opts.Name)
	if err := w.deployConf(ctx, opts, configMapName, wafConf); err != nil {
		return err
	}
	if err := w.deployNginx(ctx, opts, configMapName, wafConf); err != nil {
		return err
	}

	return c.String(http.StatusCreated, "Created nginx resource!")
}

func healthcheck(c echo.Context) error {
	return c.String(http.StatusOK, "WORKING")
}
