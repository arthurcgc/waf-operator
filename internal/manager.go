package internal

import (
	"flag"
	"path/filepath"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type WafManager struct {
	dynamicClient dynamic.Interface
	defaultClient *kubernetes.Clientset

	logger *logrus.Logger
	server *echo.Echo
}

func (w *WafManager) setRoutes() {
	w.server.POST("/deploy", w.deploy)
	w.server.GET("/healthcheck", healthcheck)
}

func (w *WafManager) buildEcho() {
	w.server = echo.New()
	w.server.Use(middleware.Logger())
	w.server.Use(middleware.Recover())

	w.setRoutes()
}

func NewInCluster() (WafManager, error) {
	mgr := WafManager{}
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return WafManager{}, err
	}
	defaultClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return WafManager{}, err
	}

	mgr.buildEcho()
	mgr.logger = logrus.New()
	mgr.defaultClient = defaultClient
	mgr.dynamicClient = dynamicClient
	return mgr, nil
}

func NewOutsideCluster() (WafManager, error) {
	mgr := WafManager{}
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
		return WafManager{}, err
	}

	// creates the clientset
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return WafManager{}, err
	}
	defaultClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return WafManager{}, err
	}

	mgr.buildEcho()
	mgr.logger = logrus.New()
	mgr.defaultClient = defaultClient
	mgr.dynamicClient = dynamicClient
	return mgr, nil
}

func (w *WafManager) Start() {
	w.logger.Fatal(w.server.Start(":8080"))
}
