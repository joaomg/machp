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
	handler struct {
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
func (h *handler) getTenant(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	tenant := h.tenants[id]
	if tenant == nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}
	return c.JSON(http.StatusOK, tenant)
}

// POST create a new tenant
func (h *handler) createTenant(c echo.Context) error {
	t := &Tenant{
		ID: seq,
	}
	if err := c.Bind(t); err != nil {
		return err
	}
	h.tenants[t.ID] = t
	seq++
	return c.JSON(http.StatusCreated, t)
}

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// data handler
	h := &handler{map[int]*Tenant{}}

	// Routes
	e.GET("/", hello)
	e.GET("/tenant/:id", h.getTenant)
	e.POST("/tenant", h.createTenant)

	e.Logger.Fatal(e.Start(":1323"))
}
