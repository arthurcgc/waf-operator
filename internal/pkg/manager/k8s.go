package manager

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/tsuru/nginx-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type k8s struct {
	dynamicClient dynamic.Interface
	defaultClient *kubernetes.Clientset
}

func NewInCluster() (Manager, error) {
	mgr := &k8s{}
	config, err := rest.InClusterConfig()
	if err != nil {
		return mgr, err
	}

	// creates the clientset
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return mgr, err
	}
	defaultClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return mgr, err
	}

	mgr.defaultClient = defaultClient
	mgr.dynamicClient = dynamicClient
	return mgr, nil
}

func NewOutsideCluster() (Manager, error) {
	mgr := &k8s{}
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return mgr, err
	}

	// creates the clientset
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return mgr, err
	}
	defaultClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return mgr, err
	}

	mgr.defaultClient = defaultClient
	mgr.dynamicClient = dynamicClient
	return mgr, nil
}

func (k *k8s) Deploy(ctx context.Context, args DeployArgs) error {
	if err := k.deployConf(ctx, args); err != nil {
		return err
	}
	if err := k.deployNginx(ctx, args); err != nil {
		return err
	}
	return nil
}

func (k *k8s) deployNginx(ctx context.Context, args DeployArgs) error {
	nginxGVR := schema.GroupVersionResource{Group: "nginx.tsuru.io", Version: "v1alpha1", Resource: "nginxes"}

	nginx := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "nginx.tsuru.io/v1alpha1",
			"kind":       "Nginx",
			"metadata": map[string]interface{}{
				"name": args.WAFName,
			},
			"spec": map[string]interface{}{
				"replicas": args.Replicas,
				"image":    viper.GetString("image"),
				"config": map[string]interface{}{
					"kind": "ConfigMap",
					"name": args.MainConfName,
				},
				"service": map[string]interface{}{
					"type": "NodePort",
				},
				// ExtraFiles references to additional files into a object in the cluster.
				// These additional files will be mounted on `/etc/nginx/extra_files`.
				"extraFiles": &v1alpha1.FilesRef{
					Name: args.WAFConfName,
					Files: map[string]string{
						"rules.conf":       "rules.conf",
						"modsecurity.conf": "modsecurity.conf",
					},
				},
			},
		},
	}

	_, err := k.dynamicClient.Resource(nginxGVR).Namespace(args.Namespace).Create(ctx, nginx, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (k *k8s) deployConf(ctx context.Context, args DeployArgs) error {
	immutable := new(bool)
	wafConf := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      args.WAFConfName,
			Namespace: args.Namespace,
		},

		Immutable: immutable,
		Data: map[string]string{
			"modsecurity.conf": recommendedConf,
			"rules.conf": `
# Include the recommended configuration
Include /etc/nginx/extra_files/modsecurity.conf
# A test rule
SecRule ARGS:testparam "@contains test" "id:1234,deny,log,status:403"
`,
		},
	}
	_, err := k.defaultClient.CoreV1().ConfigMaps(args.Namespace).Create(ctx, wafConf, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// proxy_pass http://go-hostname.backend.svc.cluster.local:80;
	mainConf := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      args.MainConfName,
			Namespace: args.Namespace,
		},

		Immutable: immutable,
		Data: map[string]string{
			"nginx.conf": fmt.Sprintf(`
	load_module modules/ngx_http_modsecurity_module.so;
	events {}

	http {
		server {
		listen 8080;
		modsecurity on;
		modsecurity_rules_file /etc/nginx/extra_files/rules.conf;

		location / {
			proxy_pass %s;
		}
	}
	}`, args.ProxyPass),
		},
	}
	_, err = k.defaultClient.CoreV1().ConfigMaps(args.Namespace).Create(ctx, mainConf, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}
