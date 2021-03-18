package api

import (
	"fmt"

	"github.com/arthurcgc/tcc/internal/pkg/manager"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Api struct {
	logger  *logrus.Logger
	server  *echo.Echo
	manager manager.Manager
}

func New() (*Api, error) {
	api := &Api{
		logger: logrus.New(),
		server: buildServer(),
	}
	var mgr manager.Manager
	var err error
	if viper.GetBool("outside_cluster") {
		mgr, err = manager.NewOutsideCluster()
		if err != nil {
			return nil, err
		}

	} else {
		mgr, err = manager.NewInCluster()
		if err != nil {
			return nil, err
		}
	}
	api.manager = mgr
	api.setRoutes()

	return api, nil
}

func (a *Api) setRoutes() {
	a.server.POST("/deploy", a.deploy)
	a.server.GET("/healthcheck", a.healthcheck)
}

func buildServer() *echo.Echo {
	server := echo.New()
	server.Use(middleware.Logger())
	server.Use(middleware.Recover())
	return server
}

func (a *Api) Start() {
	a.logger.Fatal(a.server.Start(fmt.Sprintf(":%s", viper.Get("port"))))
}
