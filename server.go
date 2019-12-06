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
		Host        string `env:"MACHP_DB_HOST" env-description:"Database host" env-default:"localhost"`
		Port        string `env:"MACHP_DB_PORT" env-description:"Database port" env-default:"3306"`
		User        string `env:"MACHP_DB_USER" env-description:"Database user name" env-default:"machp"`
		Password    string `env:"MACHP_DB_PASSWORD" env-description:"Database user password" env-default:"machp123"`
		Name        string `env:"MACHP_DB_NAME" env-description:"Database name" env-default:"machp_dev"`
		Connections int    `env:"MACHP_DB_CONNECTIONS" env-description:"Total number of database connections" env-default:"8"`
	} `yaml:"database"`
	Queue struct {
		Scheme   string `env:"MACHP_MQ_SCHENE" env-description:"Message queue scheme: amqp or amqps" env-default:"amqp"`
		User     string `env:"MACHP_MQ_USER" env-description:"Message queue user name" env-default:"guest"`
		Password string `env:"MACHP_MQ_PASSWORD" env-description:"Message queue password" env-default:"guest"`
		Host     string `env:"MACHP_MQ_HOST" env-description:"Message queue host" env-default:"localhost"`
		Port     string `env:"MACHP_MQ_PORT" env-description:"Message queue port" env-default:"5672"`
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

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func getRequestID(c echo.Context) (string, error) {
	rid := c.Request().Header.Get(echo.HeaderXRequestID)
	if rid == "" {
		rid = c.Response().Header().Get(echo.HeaderXRequestID)
	}
	return rid, nil
}

func (h *Handler) getTenantByID(id int) (Tenant, error) {
	t := &Tenant{}
	err := h.db.QueryRow("SELECT id, name, md5 FROM tenant where id = ?", id).Scan(&t.ID, &t.Name, &t.Md5)
	return *t, err
}

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
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "unable to get tenant details")
	}

	rid, _ := getRequestID(c)
	fmt.Println(rid)

	return c.JSON(http.StatusOK, t)
}

// POST create a new tenant
func (h *Handler) createTenant(c echo.Context) error {

	t := &Tenant{}
	if err := c.Bind(t); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "unable to bind context to tenant")
	}

	hash := md5.Sum([]byte(t.Name))
	t.Md5 = hex.EncodeToString(hash[:])

	stmt, err := h.db.Prepare("INSERT INTO tenant (name, md5) VALUES (?, ?)")
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusNotFound, "error preparing statement")
	}

	res, err := stmt.Exec(t.Name, t.Md5)
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
		return echo.NewHTTPError(http.StatusOK, "no change to tenant")
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

	// create tenant directory using it's first four md5 chars
	// inside the machp server home directory
	machpHome := h.cfg.Server.Home
	tenantDirectory := fmt.Sprintf("%s%c%s", machpHome, os.PathSeparator, t.Md5[:4])
	os.Mkdir(tenantDirectory, 0755)

	for _, file := range files {
		// Source
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		// Destination

		// create files inside the
		fileName := fmt.Sprintf("%s%c%s", tenantDirectory, os.PathSeparator, file.Filename)
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
	db, err := sql.Open("mysql", cfg.Database.User+":"+cfg.Database.Password+"@tcp("+cfg.Database.Host+":"+cfg.Database.Port+")/"+cfg.Database.Name)
	failOnError(err, "Failed to connect to myql database")
	defer db.Close()

	// message queue
	conn, err := amqp.Dial(cfg.Queue.Scheme + "://" + cfg.Queue.User + "@" + cfg.Queue.Host + ":" + cfg.Queue.Port + "/")
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
