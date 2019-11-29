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
	tenants    = map[int]*Tenant{}
	tenantJSON = `{"id":1,"name":"tom"}`
)

func dumpJSON(rec *httptest.ResponseRecorder) string {
	return strings.TrimSuffix(rec.Body.String(), "\n")
}

// GET test get tenant
func TestTenant(t *testing.T) {
	// Setup
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/tenant", strings.NewReader(tenantJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	d := &data{tenants}

	// createTenant
	if assert.NoError(t, d.createTenant(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, tenantJSON, dumpJSON(rec))
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetPath("/tenant/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")

	// getTenant
	if assert.NoError(t, d.getTenant(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, tenantJSON, dumpJSON(rec))
	}
}
