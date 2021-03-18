package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/arthurcgc/tcc/internal/pkg/manager"
	echo "github.com/labstack/echo/v4"
)

type DeployOpts struct {
	Name      string
	Replicas  int
	Namespace string
	ProxyPass string `json:"proxy,omitempty"`
}

func (a *Api) deploy(c echo.Context) error {
	var opts DeployOpts
	err := json.NewDecoder(c.Request().Body).Decode(&opts)
	if err != nil {
		return err
	}

	args := manager.DeployArgs{
		WAFName:      opts.Name,
		Namespace:    opts.Namespace,
		Replicas:     opts.Replicas,
		ProxyPass:    opts.ProxyPass,
		MainConfName: fmt.Sprintf("%s-conf", opts.Name),
		WAFConfName:  fmt.Sprintf("%s-conf-extra", opts.Name),
	}
	if err := a.manager.Deploy(c.Request().Context(), args); err != nil {
		return fmt.Errorf("error during deploy: %s", err.Error())
	}

	return c.String(http.StatusCreated, "Created nginx resource!\n")
}
