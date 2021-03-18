package api

import (
	"net/http"

	echo "github.com/labstack/echo/v4"
)

func (a *Api) healthcheck(c echo.Context) error {
	return c.String(http.StatusOK, "WORKING")
}
