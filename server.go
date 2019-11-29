package main

import (
	"database/sql"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type (
	// tenant type represents a tenant table row
	Tenant struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	// machp type contains a pointer to its sql.DB
	Machp struct {
		db *sql.DB
	}
)

var (
	seq = 1
)

//----------
// Handlers
//----------

// GET return a tenant
func (machp *Machp) getTenant(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	t := &Tenant{}
	err := machp.db.QueryRow("SELECT id, name FROM tenant where id = ?", id).Scan(&t.ID, &t.Name)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "unable to query tenant")
	}

	return c.JSON(http.StatusOK, t)
}

// POST create a new tenant
func (machp *Machp) createTenant(c echo.Context) error {

	t := &Tenant{}
	if err := c.Bind(t); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "unable to bind context to tenant")
	}

	stmt, err := machp.db.Prepare("INSERT INTO tenant (name) VALUES (?)")
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "error preparing statement")
	}

	res, err := stmt.Exec(t.Name)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "error executing statement")
	}

	id, err := res.LastInsertId()
	err = machp.db.QueryRow("SELECT id, name FROM tenant where id = ?", id).Scan(&t.ID, &t.Name)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "unable to query tenant")
	}

	return c.JSON(http.StatusCreated, t)
}

// DELETE a tenant
func (machp *Machp) deleteTenant(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	rows, err := machp.db.Query("DELETE FROM tenant WHERE id = ?", id)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "error deleting tenant")
	}
	defer rows.Close()

	return c.NoContent(http.StatusNoContent)
}

// PUT update tenant details
func (machp *Machp) putTenant(c echo.Context) error {
	t := new(Tenant)
	if err := c.Bind(t); err != nil {
		return err
	}

	rows, err := machp.db.Query("UPDATE tenant SET name = ? WHERE id = ?", t.Name, t.ID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "error updating tenant")
	}
	defer rows.Close()

	return c.JSON(http.StatusOK, t)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Machp
	db, err := sql.Open("mysql", "machp:machp123@tcp(localhost:3306)/machp_dev")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	machp := &Machp{db}

	// Routes
	e.GET("/tenant/:id", machp.getTenant)
	e.POST("/tenant", machp.createTenant)
	e.DELETE("/tenant/:id", machp.deleteTenant)
	e.PUT("/tenant/:id", machp.putTenant)

	e.Logger.Fatal(e.Start(":1323"))
}
