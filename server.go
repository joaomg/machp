package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
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
		Md5  string `json:"md5"`
	}
	// handler type contains a pointer to its sql.DB
	Handler struct {
		db *sql.DB
	}
)

var (
	seq = 1
)

//----------
// Handlers
//----------

func (h *Handler) getTenantByID(id int) (Tenant, error) {
	t := &Tenant{}
	err := h.db.QueryRow("SELECT id, name FROM tenant where id = ?", id).Scan(&t.ID, &t.Name)
	return *t, err
}

func (h *Handler) getTenantByName(name string) (Tenant, error) {
	t := &Tenant{}
	err := h.db.QueryRow("SELECT id, name FROM tenant where name = ?", name).Scan(&t.ID, &t.Name)
	return *t, err
}

// GET return a tenant
func (h *Handler) getTenant(c echo.Context) error {
	// fetch tenant id from route param
	id, _ := strconv.Atoi(c.Param("id"))

	t, err := h.getTenantByID(id)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "unable to get tenant details")
	}

	return c.JSON(http.StatusOK, t)
}

// POST create a new tenant
func (h *Handler) createTenant(c echo.Context) error {

	t := &Tenant{}
	if err := c.Bind(t); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "unable to bind context to tenant")
	}

	stmt, err := h.db.Prepare("INSERT INTO tenant (name) VALUES (?)")
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
	err = h.db.QueryRow("SELECT id, name FROM tenant where id = ?", id).Scan(&t.ID, &t.Name)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "unable to query tenant")
	}

	return c.JSON(http.StatusCreated, t)
}

// DELETE a tenant
func (h *Handler) deleteTenant(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	rows, err := h.db.Query("DELETE FROM tenant WHERE id = ?", id)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "error deleting tenant")
	}
	defer rows.Close()

	return c.NoContent(http.StatusNoContent)
}

// PUT update tenant details
func (h *Handler) putTenant(c echo.Context) error {

	// bint tenant using the request JSON
	t := new(Tenant)
	if err := c.Bind(t); err != nil {
		return err
	}

	// change tenant details
	res, err := h.db.Exec("UPDATE tenant SET name = ? WHERE id = ?", t.Name, t.ID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "error updating tenant")
	}

	// check if the tenant was changed
	count, _ := res.RowsAffected()
	if count == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "unable to update tenant")
	}

	return c.JSON(http.StatusOK, t)
}

// UPLOAD file to tenant
func (h *Handler) uploadToTenant(c echo.Context) error {
	// fetch tenant name from route param
	name := c.Param("name")

	t, err := h.getTenantByName(name)
	c.Logger().Info(t)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "unable to get tenant details")
	}

	//------------
	// Read files
	//------------

	// Multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return err
	}
	files := form.File["files"]

	for _, file := range files {
		// Source
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		// Destination

		// tenant directory home using it's name
		os.Mkdir(t.Name, 0755)

		// create files inside the
		fileName := fmt.Sprintf("%s%c%s", t.Name, os.PathSeparator, file.Filename)
		dst, err := os.Create(fileName)
		if err != nil {
			return err
		}
		defer dst.Close()

		// Copy
		if _, err = io.Copy(dst, src); err != nil {
			return err
		}

	}

	return c.JSON(http.StatusOK, files)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))
	e.Use(middleware.Recover())

	// Handler
	db, err := sql.Open("mysql", "machp:machp123@tcp(localhost:3306)/machp_dev")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	machp := &Handler{db}

	// Routes
	e.GET("/tenant/:id", machp.getTenant)
	e.POST("/tenant", machp.createTenant)
	e.DELETE("/tenant/:id", machp.deleteTenant)
	e.PUT("/tenant/:id", machp.putTenant)
	e.POST("/tenant/:name/upload", machp.uploadToTenant)

	e.Logger.Fatal(e.Start(":1323"))
}
