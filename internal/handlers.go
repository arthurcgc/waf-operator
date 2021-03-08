package internal

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (w *WafManager) deploy(c echo.Context) error {
	nginxGVR := schema.GroupVersionResource{Group: "nginx.tsuru.io", Version: "v1alpha1", Resource: "nginx"}

	nginx := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "nginx.tsuru.io/v1alpha1",
			"kind":       "Nginx",
			"metadata": map[string]interface{}{
				"name": "waf2",
			},
			"spec": map[string]interface{}{
				"replicas": 1,
				"config": map[string]interface{}{
					"kind": "ConfigMap",
					"name": "nginxconf",
				},
				"service": map[string]interface{}{
					"type": "NodePort",
				},
			},
		},
	}

	fmt.Println("Creating nginx...")
	_, err := w.dynamicClient.Resource(nginxGVR).Namespace("frontend").Create(context.TODO(), nginx, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// 	immutable := new(bool)
	// 	configMap := &v1.ConfigMap{
	// 		metav1.TypeMeta{
	// 			Kind:       "ConfigMap",
	// 			APIVersion: "v1",
	// 		},
	// 		metav1.ObjectMeta{
	// 			Name:      "nginxWAFconf",
	// 			Namespace: "frontend",
	// 		},

	// 		immutable,
	// 		map[string]string{
	// 			"nginx.conf": `
	// events {}

	// http {
	// 	server {
	// 	listen 8080;

	// 	location / {
	// 		proxy_pass http://go-hostname.backend.svc.cluster.local:80;
	// 	}
	// }
	// }`,
	// 		},
	// 		nil,
	// 	}
	// _, err = w.defaultClient.CoreV1().ConfigMaps("frontend").Create(context.TODO(), configMap, metav1.CreateOptions{})
	// if err != nil {
	// 	return err
	// }

	return c.String(http.StatusCreated, "Created nginx resource!")
}

func healthcheck(c echo.Context) error {
	return c.String(http.StatusOK, "WORKING")
}
