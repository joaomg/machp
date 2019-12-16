package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/streadway/amqp"
)

// Config is a application configuration structure
type Config struct {
	Database struct {
		Driver     string `env:"MACHP_DB_DRIVER" env-description:"Database driver" env-default:"mysql"`
		DataSource string `env:"MACHP_DB_DATA_SOURCE" env-description:"Database data source" env-default:"machp:machp123@tcp(localhost:3306)/machp_dev"`
	} `yaml:"database"`
	Queue struct {
		URL string `env:"MACHP_MQ_URL" env-description:"Message queue AMPQ uri string" env-default:"amqp://guest@localhost:5672/"`
	} `yaml:"queue"`
	Server struct {
		Host string `env:"MACHP_HOST" env-description:"Server host" env-default:"localhost"`
		Port string `env:"MACHP_PORT" env-description:"Server port" env-default:"1323"`
		Home string `env:"MACHP_HOME" env-description:"Server home directory" env-default:"files"`
	} `yaml:"server"`
}

// Handler, Tenant types
type (
	// handler type contains a pointer to its sql.DB
	Handler struct {
		db              *sql.DB
		cfg             *Config
		producerChannel *amqp.Channel
		consumerChannel *amqp.Channel
	}
	// tenant type represents a tenant table row
	Tenant struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Md5  string `json:"md5"`
	}
)

// terminate server execution and log fatal error
func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

// if there's an error return HTTPError
func returnOnError(c echo.Context, err error, httpStatus int, msg string) *echo.HTTPError {
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(httpStatus, msg)
	}
	return nil
}

// generate the request ID
func getRequestID(c echo.Context) (string, error) {
	rid := c.Request().Header.Get(echo.HeaderXRequestID)
	if rid == "" {
		rid = c.Response().Header().Get(echo.HeaderXRequestID)
	}
	return rid, nil
}

// return Tenant from database using the id
func (h *Handler) getTenantByID(id int) (Tenant, error) {
	t := &Tenant{}
	err := h.db.QueryRow("SELECT id, name, md5 FROM tenant where id = ?", id).Scan(&t.ID, &t.Name, &t.Md5)
	return *t, err
}

// return Tenant from database using the name
func (h *Handler) getTenantByName(name string) (Tenant, error) {
	t := &Tenant{}
	err := h.db.QueryRow("SELECT id, name, md5 FROM tenant where name = ?", name).Scan(&t.ID, &t.Name, &t.Md5)
	return *t, err
}

// GET return a tenant
func (h *Handler) getTenant(c echo.Context) error {
	// fetch tenant id from route param
	id, _ := strconv.Atoi(c.Param("id"))

	t, err := h.getTenantByID(id)
	returnOnError(c, err, http.StatusNotFound, "Unable to get tenant details")

	rid, _ := getRequestID(c)
	fmt.Println(rid)

	return c.JSON(http.StatusOK, t)
}

// POST create a new tenant
func (h *Handler) createTenant(c echo.Context) error {

	t := &Tenant{}
	err := c.Bind(t)
	returnOnError(c, err, http.StatusNotFound, "Unable to bind context to tenant")

	hash := md5.Sum([]byte(t.Name))
	t.Md5 = hex.EncodeToString(hash[:])

	stmt, err := h.db.Prepare("INSERT INTO tenant (name, md5) VALUES (?, ?)")
	returnOnError(c, err, http.StatusInternalServerError, "Error preparing statement")

	res, err := stmt.Exec(t.Name, t.Md5)
	returnOnError(c, err, http.StatusInternalServerError, "Error executing statement")

	id, err := res.LastInsertId()
	err = h.db.QueryRow("SELECT id, name FROM tenant where id = ?", id).Scan(&t.ID, &t.Name)
	returnOnError(c, err, http.StatusInternalServerError, "Unable to query tenant")

	return c.JSON(http.StatusCreated, t)
}

// DELETE a tenant
func (h *Handler) deleteTenant(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	_, err := h.db.Query("DELETE FROM tenant WHERE id = ?", id)
	returnOnError(c, err, http.StatusInternalServerError, "Error deleting tenant")

	return c.NoContent(http.StatusNoContent)
}

// PUT update tenant details
func (h *Handler) putTenant(c echo.Context) error {

	// bint tenant using the request JSON
	t := new(Tenant)
	err := c.Bind(t)
	returnOnError(c, err, http.StatusNotFound, "Unable to bind context to tenant")

	// change tenant details
	res, err := h.db.Exec("UPDATE tenant SET name = ? WHERE id = ?", t.Name, t.ID)
	returnOnError(c, err, http.StatusInternalServerError, "error updating tenant")

	// check if the tenant was changed
	count, _ := res.RowsAffected()
	if count == 0 {
		return echo.NewHTTPError(http.StatusOK, "No change to tenant")
	}

	// get tenant from db
	changedTenant, _ := h.getTenantByID(t.ID)

	return c.JSON(http.StatusOK, changedTenant)
}

// UPLOAD file to tenant
func (h *Handler) uploadToTenant(c echo.Context) error {
	// fetch tenant name from route param
	name := c.Param("name")

	t, err := h.getTenantByName(name)
	returnOnError(c, err, http.StatusNotFound, "Unable to get tenant details")

	//------------
	// Read files
	//------------

	// Multipart form
	form, err := c.MultipartForm()
	returnOnError(c, err, http.StatusInternalServerError, "Error accessing multipart form")
	files := form.File["files"]

	// create tenant directory using it's first four md5 chars
	// inside the machp server home directory
	machpHome := h.cfg.Server.Home
	tenantDirectory := fmt.Sprintf("%s%c%s", machpHome, os.PathSeparator, t.Md5[:4])
	os.Mkdir(tenantDirectory, 0755)

	for _, file := range files {
		// Source
		src, err := file.Open()
		returnOnError(c, err, http.StatusInternalServerError, "Error opening file")
		defer src.Close()

		// Destination

		// create files inside the
		fileName := fmt.Sprintf("%s%c%s", tenantDirectory, os.PathSeparator, file.Filename)
		dst, err := os.Create(fileName)
		returnOnError(c, err, http.StatusNotFound, "Error creating file")
		defer dst.Close()

		// Copy
		_, err = io.Copy(dst, src)
		returnOnError(c, err, http.StatusNotFound, "Error saving file")
	}

	return c.JSON(http.StatusOK, files)
}

func main() {
	// Config
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Print("Failed to read configuration from environment")
		panic(err.Error())
	}

	// Init

	// file system
	machpHome := cfg.Server.Home
	os.Mkdir(machpHome, 0755)

	// database
	db, err := sql.Open(cfg.Database.Driver, cfg.Database.DataSource)
	failOnError(err, "Failed to connect to myql database")
	defer db.Close()

	// message queue
	conn, err := amqp.Dial(cfg.Queue.URL)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	pch, err := conn.Channel()
	failOnError(err, "Failed to open a RabbitMQ channel")
	defer pch.Close()

	cch, err := conn.Channel()
	failOnError(err, "Failed to open a RabbitMQ channel")
	defer cch.Close()

	// Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.RequestID())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}, requestid=${id}\n",
	}))
	e.Use(middleware.Recover())

	// Handler
	machp := &Handler{db, &cfg, pch, cch}

	// Routes
	e.GET("/tenant/:id", machp.getTenant)
	e.POST("/tenant", machp.createTenant)
	e.DELETE("/tenant/:id", machp.deleteTenant)
	e.PUT("/tenant/:id", machp.putTenant)
	e.POST("/tenant/:name/upload", machp.uploadToTenant)

	// Start Echo
	e.Logger.Fatal(e.Start(cfg.Server.Host + ":" + cfg.Server.Port))
}
