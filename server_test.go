package main

import (
	"bytes"
	"database/sql"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

var (
	driver          = "mysql"
	dataSource      = "machp:machp123@tcp(localhost:3306)/machp_dev"
	tenantJSONTom   = `{"id":1,"name":"tom","md5":"34b7da764b21d298ef307d04d8152dc5"}`
	tenantJSONJerry = `{"id":1,"name":"jerry","md5":"34b7da764b21d298ef307d04d8152dc5"}`
)

func dumpJSON(rec *httptest.ResponseRecorder) string {
	return strings.TrimSuffix(rec.Body.String(), "\n")
}

// test drop and create schema
// creates an empty schema suitable for testing
func TestSchema(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "mysql", "-v", "-hlocalhost", "-P3306", "-umachp", "-pmachp123", "machp_dev", "<", "1_machp_schema.sql")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	assert.Nil(t, err)
}

// remove and create local file system
// remove the files directory
// create the files directory
func TestFileSystem(t *testing.T) {

	// delete files directory
	assert.Nil(t, os.RemoveAll("files"))

	// create files directory
	assert.Nil(t, os.Mkdir("files", 0755))
}

// test tenant get, create, update and delete
func TestTenant(t *testing.T) {

	// Handler
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Print("Failed to read configuration from environment")
		panic(err.Error())
	}

	db, err := sql.Open(driver, dataSource)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	h := &Handler{db, &cfg, nil, nil}

	// Setup
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/tenant", strings.NewReader(tenantJSONTom))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// createTenant tom
	if assert.NoError(t, h.createTenant(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, tenantJSONTom, dumpJSON(rec))
	}

	// putTenant change tenant 1 name from Tom to Jerry
	req = httptest.NewRequest(http.MethodPut, "/tenant", strings.NewReader(tenantJSONJerry))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetPath("/tenant/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")

	if assert.NoError(t, h.putTenant(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, tenantJSONJerry, dumpJSON(rec))
	}

	// getTenant return tenant 1 named Jerry
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetPath("/tenant/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")

	if assert.NoError(t, h.getTenant(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, tenantJSONJerry, dumpJSON(rec))
	}

	// uploadToTenant
	path := "test/abc.txt"
	file, err := os.Open(path)
	assert.Nil(t, err)
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", filepath.Base(path))
	assert.Nil(t, err)
	_, err = io.Copy(part, file)
	writer.Close()

	req = httptest.NewRequest(http.MethodPost, "/tenant/:name/upload", body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetPath("/tenant/:name/upload")
	c.SetParamNames("name")
	c.SetParamValues("jerry")

	if assert.NoError(t, h.uploadToTenant(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	_, err = os.Stat("files/34b7/abc.txt")
	assert.NoError(t, err)
	assert.True(t, !os.IsNotExist(err))

	// deleteTenant delete the tenant with id 1
	req = httptest.NewRequest(http.MethodDelete, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetPath("/tenant/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")

	if assert.NoError(t, h.deleteTenant(c)) {
		assert.Equal(t, http.StatusNoContent, rec.Code)
	}

}
