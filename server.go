package main

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type (
	Tenant struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	data struct {
		tenants map[int]*Tenant
	}
)

var (
	seq = 1
)

//----------
// Handlers
//----------

// GET return a tenant
func (d *data) getTenant(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	tenant := d.tenants[id]
	if tenant == nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}
	return c.JSON(http.StatusOK, tenant)
}

// POST create a new tenant
func (d *data) createTenant(c echo.Context) error {
	t := &Tenant{
		ID: seq,
	}
	if err := c.Bind(t); err != nil {
		return err
	}
	d.tenants[t.ID] = t
	seq++
	return c.JSON(http.StatusCreated, t)
}

// DELETE a tenant
func (d *data) deleteTenant(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	tenant := d.tenants[id]
	if tenant == nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}
	delete(d.tenants, id)
	return c.NoContent(http.StatusNoContent)
}

// PUT update tenant details
func (d *data) putTenant(c echo.Context) error {
	t := new(Tenant)
	if err := c.Bind(t); err != nil {
		return err
	}
	id, _ := strconv.Atoi(c.Param("id"))
	d.tenants[id] = t
	return c.JSON(http.StatusOK, d.tenants[id])
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// data handler
	d := &data{map[int]*Tenant{}}

	// Routes
	e.GET("/tenant/:id", d.getTenant)
	e.POST("/tenant", d.createTenant)
	e.DELETE("/tenant/:id", d.deleteTenant)
	e.PUT("/tenant/:id", d.putTenant)

	e.Logger.Fatal(e.Start(":1323"))
}
