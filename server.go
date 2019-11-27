package main

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type (
	Tenant struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
)

var (
	tenants = map[int]*Tenant{}
	seq     = 1
)

//----------
// Handlers
//----------

// POST create a new tenant
func createTenant(c echo.Context) error {
	t := &Tenant{
		ID: seq,
	}
	if err := c.Bind(t); err != nil {
		return err
	}
	tenants[t.ID] = t
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

	// Routes
	e.GET("/", hello)
	e.POST("/tenant", createTenant)

	e.Logger.Fatal(e.Start(":1323"))
}
