package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

var (
	machpTest       = "machp:machp123@tcp(localhost:3306)/machp_dev"
	tenantJSONTom   = `{"id":1,"name":"tom"}`
	tenantJSONJerry = `{"id":1,"name":"jerry"}`
)

func dumpJSON(rec *httptest.ResponseRecorder) string {
	return strings.TrimSuffix(rec.Body.String(), "\n")
}

// test tenant get, create, update and delete
func TestTenant(t *testing.T) {
	// Setup
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/tenant", strings.NewReader(tenantJSONTom))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db, err := sql.Open("mysql", machpTest)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	m := &Handler{db}

	// createTenant tom
	if assert.NoError(t, m.createTenant(c)) {
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

	if assert.NoError(t, m.putTenant(c)) {
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

	if assert.NoError(t, m.getTenant(c)) {
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

	if assert.NoError(t, m.deleteTenant(c)) {
		assert.Equal(t, http.StatusNoContent, rec.Code)
	}

}
