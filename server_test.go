package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

var (
	tenants         = map[int]*Tenant{}
	tenantJSONTom   = `{"id":1,"name":"tom"}`
	tenantJSONJerry = `{"id":1,"name":"jerry"}`
)

func dumpJSON(rec *httptest.ResponseRecorder) string {
	return strings.TrimSuffix(rec.Body.String(), "\n")
}

// GET test get tenant
func TestTenant(t *testing.T) {
	// Setup
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/tenant", strings.NewReader(tenantJSONTom))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	d := &data{tenants}

	// createTenant tom
	if assert.NoError(t, d.createTenant(c)) {
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

	if assert.NoError(t, d.putTenant(c)) {
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

	if assert.NoError(t, d.getTenant(c)) {
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

	if assert.NoError(t, d.deleteTenant(c)) {
		assert.Equal(t, http.StatusNoContent, rec.Code)
	}

}
