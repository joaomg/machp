package main

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

var (
	machpTest       = "machp:machp123@tcp(localhost:3306)/machp_dev"
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

// test tenant get, create, update and delete
func TestTenant(t *testing.T) {
	// Setup
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/tenant", strings.NewReader(tenantJSONTom))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var cfg Config
	db, err := sql.Open("mysql", machpTest)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	h := &Handler{db, &cfg}

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
