// SAP OData Mock Server
//
// Simulates a SAP NetWeaver OData service for local development and integration
// testing. Every incoming HTTP request is printed to stdout with full detail:
// method, URL, headers, query parameters, body and — where applicable — the
// parsed OData entity-set name and key predicate.
//
// Simulated endpoints (all require Basic-Auth unless -noauth is passed):
//
//	GET  /sap/opu/odata/sap/UI_PRODUCTLIST/$metadata
//	GET  /sap/opu/odata/sap/UI_PRODUCTLIST/ProductList
//	GET  /sap/opu/odata/sap/UI_PRODUCTLIST/ProductList(Product='X',Plant='Y',ValuationType='Z')
//	GET  /sap/opu/odata/sap/API_COMPANYCODE_SRV/$metadata
//	GET  /sap/opu/odata/sap/API_COMPANYCODE_SRV/A_CompanyCode
//	GET  /sap/opu/odata/sap/API_COMPANYCODE_SRV/A_CompanyCode('code')
//	GET  /sap/opu/odata/sap/SD_SALES_ORDER_IMPORT/$metadata
//	GET  /sap/opu/odata/sap/SD_SALES_ORDER_IMPORT/I_SalesOrderImport
//	GET  /sap/opu/odata/sap/API_MAINTNOTIFICATION/$metadata  (CSRF fetch)
//	POST /sap/opu/odata/sap/API_MAINTNOTIFICATION/$metadata  (CSRF fetch)
//	POST /sap/opu/odata/sap/API_MAINTNOTIFICATION/MaintenanceNotification
//
// CSRF flow:
//
//	1. Client sends GET/POST to $metadata with header  X-CSRF-Token: Fetch
//	2. Server responds with a one-time token in header  X-CSRF-Token: <token>
//	3. Client attaches that token to the mutation request
//	4. Server validates it; on mismatch returns 403
//
// Usage:
//
//	go run . [flags]
//
//	-addr  string  listen address (default ":44300")
//	-user  string  expected Basic-Auth username (default "sapuser")
//	-pass  string  expected Basic-Auth password (default "sappass")
//	-noauth        disable Basic-Auth check
//	-nocolor       disable ANSI color in log output
//	-delay int     artificial response delay in milliseconds (default 0)
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── CLI flags ────────────────────────────────────────────────────────────────

var (
	addr    = flag.String("addr", ":44300", "listen address")
	user    = flag.String("user", "sapuser", "Basic-Auth username")
	pass    = flag.String("pass", "sappass", "Basic-Auth password")
	noAuth  = flag.Bool("noauth", false, "disable Basic-Auth validation")
	noColor = flag.Bool("nocolor", false, "disable ANSI color output")
	delay   = flag.Int("delay", 0, "artificial response delay (ms)")
)

// ─── ANSI helpers ─────────────────────────────────────────────────────────────

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	cyan   = "\033[36m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	blue   = "\033[34m"
	gray   = "\033[90m"
	purple = "\033[35m"
)

func color(c, s string) string {
	if *noColor {
		return s
	}
	return c + s + reset
}

// ─── CSRF token store ─────────────────────────────────────────────────────────

type csrfStore struct {
	mu     sync.Mutex
	tokens map[string]time.Time // token → issued-at
}

func newCSRFStore() *csrfStore { return &csrfStore{tokens: make(map[string]time.Time)} }

func (s *csrfStore) issue() string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	tok := base64.RawURLEncoding.EncodeToString(b)
	s.mu.Lock()
	// purge tokens older than 10 minutes
	for k, t := range s.tokens {
		if time.Since(t) > 10*time.Minute {
			delete(s.tokens, k)
		}
	}
	s.tokens[tok] = time.Now()
	s.mu.Unlock()
	return tok
}

func (s *csrfStore) validate(tok string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tokens[tok]
	if !ok {
		return false
	}
	delete(s.tokens, tok) // one-time use
	return time.Since(t) <= 10*time.Minute
}

// ─── Sample data ──────────────────────────────────────────────────────────────

var products = []map[string]interface{}{
	{"Product": "3001008", "Plant": "1010", "ValuationType": "STANDARD", "ProductType": "FERT", "InventoryPrice": "125.50", "Currency": "PEN", "PriceUnitQty": "1"},
	{"Product": "3001009", "Plant": "1010", "ValuationType": "STANDARD", "ProductType": "HALB", "InventoryPrice": "88.00", "Currency": "PEN", "PriceUnitQty": "1"},
	{"Product": "3001010", "Plant": "1020", "ValuationType": "STANDARD", "ProductType": "ROH", "InventoryPrice": "12.75", "Currency": "USD", "PriceUnitQty": "1"},
	{"Product": "3001011", "Plant": "1020", "ValuationType": "MAP", "ProductType": "FERT", "InventoryPrice": "200.00", "Currency": "PEN", "PriceUnitQty": "10"},
	{"Product": "3001012", "Plant": "1030", "ValuationType": "STANDARD", "ProductType": "DIEN", "InventoryPrice": "0.00", "Currency": "PEN", "PriceUnitQty": "1"},
}

var companyCodes = []map[string]interface{}{
	{"CompanyCode": "1000", "CompanyCodeName": "Alicorp SAA", "CityName": "Lima", "Currency": "PEN", "Country": "PE"},
	{"CompanyCode": "2000", "CompanyCodeName": "Alicorp Bolivia", "CityName": "La Paz", "Currency": "BOB", "Country": "BO"},
}

var salesOrders = []map[string]interface{}{
	{"SalesOrder": "0000100001", "SalesOrderType": "OR", "CreatedByUser": "JFERNANDEZ", "TotalNetAmount": "1500.00", "TransactionCurrency": "PEN"},
	{"SalesOrder": "0000100002", "SalesOrderType": "OR", "CreatedByUser": "MLOPEZ", "TotalNetAmount": "750.00", "TransactionCurrency": "PEN"},
	{"SalesOrder": "0000100003", "SalesOrderType": "RE", "CreatedByUser": "JFERNANDEZ", "TotalNetAmount": "-200.00", "TransactionCurrency": "USD"},
}

var notifCounter = 10000001

// ─── OData v2 response wrappers ───────────────────────────────────────────────

func v2List(items []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"d": map[string]interface{}{"results": items}}
}

func v2Entity(item map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"d": item}
}

// ─── Embedded EDMX metadata ───────────────────────────────────────────────────

const productMetadata = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx"
           xmlns:sap="http://www.sap.com/Ext/EP/AP/ProcessComponent/V1">
  <edmx:DataServices m:DataServiceVersion="2.0"
    xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="UI_PRODUCTLIST" xml:lang="en"
      xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="ProductListType" sap:content-version="1">
        <Key><PropertyRef Name="Product"/><PropertyRef Name="Plant"/><PropertyRef Name="ValuationType"/></Key>
        <Property Name="Product"        Type="Edm.String" Nullable="false" MaxLength="18" sap:label="Material"/>
        <Property Name="Plant"          Type="Edm.String" Nullable="false" MaxLength="4"  sap:label="Plant"/>
        <Property Name="ValuationType"  Type="Edm.String" Nullable="false" MaxLength="10" sap:label="Valuation Type"/>
        <Property Name="ProductType"    Type="Edm.String" Nullable="true"  MaxLength="4"  sap:label="Material Type"/>
        <Property Name="InventoryPrice" Type="Edm.Decimal" Nullable="true" Precision="23" Scale="3" sap:label="Inventory Price"/>
        <Property Name="Currency"       Type="Edm.String" Nullable="true"  MaxLength="5"  sap:label="Currency"/>
        <Property Name="PriceUnitQty"   Type="Edm.Decimal" Nullable="true" Precision="13" Scale="3" sap:label="Price Unit"/>
      </EntityType>
      <EntityContainer Name="UI_PRODUCTLIST_Entities" m:IsDefaultEntityContainer="true">
        <EntitySet Name="ProductList" EntityType="UI_PRODUCTLIST.ProductListType" sap:addressable="true"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

const companyMetadata = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
  <edmx:DataServices m:DataServiceVersion="2.0"
    xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="API_COMPANYCODE_SRV" xml:lang="en"
      xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="A_CompanyCodeType">
        <Key><PropertyRef Name="CompanyCode"/></Key>
        <Property Name="CompanyCode"     Type="Edm.String" Nullable="false" MaxLength="4"/>
        <Property Name="CompanyCodeName" Type="Edm.String" Nullable="true"  MaxLength="25"/>
        <Property Name="CityName"        Type="Edm.String" Nullable="true"  MaxLength="25"/>
        <Property Name="Currency"        Type="Edm.String" Nullable="true"  MaxLength="5"/>
        <Property Name="Country"         Type="Edm.String" Nullable="true"  MaxLength="3"/>
      </EntityType>
      <EntityContainer Name="API_COMPANYCODE_SRV_Entities" m:IsDefaultEntityContainer="true">
        <EntitySet Name="A_CompanyCode" EntityType="API_COMPANYCODE_SRV.A_CompanyCodeType"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

const salesOrderMetadata = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
  <edmx:DataServices m:DataServiceVersion="2.0"
    xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="SD_SALES_ORDER_IMPORT" xml:lang="en"
      xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="I_SalesOrderImportType">
        <Key><PropertyRef Name="SalesOrder"/></Key>
        <Property Name="SalesOrder"          Type="Edm.String"  Nullable="false" MaxLength="10"/>
        <Property Name="SalesOrderType"      Type="Edm.String"  Nullable="true"  MaxLength="4"/>
        <Property Name="CreatedByUser"        Type="Edm.String"  Nullable="true"  MaxLength="12"/>
        <Property Name="TotalNetAmount"       Type="Edm.Decimal" Nullable="true"  Precision="16" Scale="3"/>
        <Property Name="TransactionCurrency" Type="Edm.String"  Nullable="true"  MaxLength="5"/>
      </EntityType>
      <EntityContainer Name="SD_SALES_ORDER_IMPORT_Entities" m:IsDefaultEntityContainer="true">
        <EntitySet Name="I_SalesOrderImport" EntityType="SD_SALES_ORDER_IMPORT.I_SalesOrderImportType"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

const notifMetadata = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
  <edmx:DataServices m:DataServiceVersion="2.0"
    xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="API_MAINTNOTIFICATION" xml:lang="en"
      xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="MaintenanceNotificationType">
        <Key><PropertyRef Name="MaintenanceNotification"/></Key>
        <Property Name="MaintenanceNotification"      Type="Edm.String" Nullable="false" MaxLength="12"/>
        <Property Name="NotificationType"             Type="Edm.String" Nullable="false" MaxLength="2"/>
        <Property Name="NotificationText"             Type="Edm.String" Nullable="true"  MaxLength="40"/>
        <Property Name="MaintPriority"                Type="Edm.String" Nullable="true"  MaxLength="1"/>
        <Property Name="MaintNotificationCatalog"     Type="Edm.String" Nullable="true"  MaxLength="1"/>
        <Property Name="MaintNotificationCode"        Type="Edm.String" Nullable="true"  MaxLength="4"/>
        <Property Name="MaintNotificationCodeGroup"   Type="Edm.String" Nullable="true"  MaxLength="8"/>
        <Property Name="CatalogProfile"               Type="Edm.String" Nullable="true"  MaxLength="2"/>
        <Property Name="MaintObjDowntimeDurationUnit" Type="Edm.String" Nullable="true"  MaxLength="3"/>
        <Property Name="MaintObjectDowntimeDuration"  Type="Edm.Decimal" Nullable="true" Precision="13" Scale="3"/>
        <Property Name="MaintenanceObjectIsDown"      Type="Edm.Boolean" Nullable="true"/>
        <Property Name="RequiredStartDate"            Type="Edm.DateTime" Nullable="true"/>
        <Property Name="RequiredStartTime"            Type="Edm.Time"     Nullable="true"/>
      </EntityType>
      <EntityContainer Name="API_MAINTNOTIFICATION_Entities" m:IsDefaultEntityContainer="true">
        <EntitySet Name="MaintenanceNotification" EntityType="API_MAINTNOTIFICATION.MaintenanceNotificationType"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

// ─── Request logger ───────────────────────────────────────────────────────────

var reqCounter int64
var reqMu sync.Mutex

func nextReqID() string {
	reqMu.Lock()
	reqCounter++
	id := reqCounter
	reqMu.Unlock()
	return fmt.Sprintf("#%04d", id)
}

func logRequest(r *http.Request, body []byte) {
	id := nextReqID()
	sep := strings.Repeat("─", 80)

	fmt.Println()
	fmt.Println(color(cyan, sep))

	// ── Method + URL ──────────────────────────────────────────────────────────
	methodColor := green
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		methodColor = yellow
	case http.MethodDelete:
		methodColor = red
	}
	fmt.Printf("%s  %s  %s\n",
		color(gray, id),
		color(methodColor+bold, r.Method),
		color(bold, r.URL.RequestURI()),
	)
	fmt.Printf("   %s  %s  %s\n",
		color(gray, "Proto:"),
		r.Proto,
		color(gray, fmt.Sprintf("  RemoteAddr: %s", r.RemoteAddr)),
	)
	fmt.Println(color(cyan, strings.Repeat("─", 40)))

	// ── Auth ──────────────────────────────────────────────────────────────────
	u, p, hasBasic := r.BasicAuth()
	if hasBasic {
		fmt.Printf("   %s  user=%s  pass=%s\n",
			color(purple, "BasicAuth:"),
			color(bold, u),
			color(gray, maskPass(p)),
		)
	}

	// ── Headers ───────────────────────────────────────────────────────────────
	fmt.Printf("   %s\n", color(blue, "Headers:"))
	for k, vs := range r.Header {
		for _, v := range vs {
			switch strings.ToLower(k) {
			case "authorization":
				v = "[redacted]"
			case "x-csrf-token":
				v = color(yellow, v)
			}
			fmt.Printf("      %-35s %s\n", color(gray, k+":"), v)
		}
	}

	// ── Query parameters ──────────────────────────────────────────────────────
	q := r.URL.Query()
	if len(q) > 0 {
		fmt.Printf("   %s\n", color(blue, "Query Params:"))
		for k, vs := range q {
			fmt.Printf("      %-30s %s\n", color(gray, k+"="), strings.Join(vs, ", "))
		}
	}

	// ── OData path analysis ───────────────────────────────────────────────────
	if entitySet, keyPred, ok := parseODataPath(r.URL.Path); ok {
		fmt.Printf("   %s\n", color(blue, "OData Path:"))
		fmt.Printf("      %-20s %s\n", color(gray, "EntitySet:"), color(bold, entitySet))
		if keyPred != "" {
			fmt.Printf("      %-20s %s\n", color(gray, "KeyPredicate:"), color(yellow, keyPred))
		}
	}

	// ── Body ──────────────────────────────────────────────────────────────────
	if len(body) > 0 {
		fmt.Printf("   %s  (%d bytes)\n", color(blue, "Body:"), len(body))
		// Pretty-print JSON
		if isJSON := json.Valid(body); isJSON {
			var pretty interface{}
			if err := json.Unmarshal(body, &pretty); err == nil {
				bs, _ := json.MarshalIndent(pretty, "      ", "  ")
				fmt.Printf("      %s\n", color(gray, string(bs)))
			}
		} else {
			fmt.Printf("      %s\n", color(gray, truncate(string(body), 512)))
		}
	}

	fmt.Println(color(cyan, sep))
	fmt.Println()
}

func logResponse(statusCode int, headers http.Header, note string) {
	statusColor := green
	if statusCode >= 400 {
		statusColor = red
	} else if statusCode >= 300 {
		statusColor = yellow
	}
	fmt.Printf("   %s %s  %s\n",
		color(gray, "→ Response:"),
		color(statusColor+bold, strconv.Itoa(statusCode)),
		color(gray, note),
	)
	for k, vs := range headers {
		if strings.ToLower(k) == "x-csrf-token" {
			fmt.Printf("      %-35s %s\n", color(gray, k+":"), color(yellow, strings.Join(vs, ", ")))
		}
	}
	fmt.Println()
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

var odataKeyRe = regexp.MustCompile(`([^/]+)\(([^)]+)\)$`)
var odataSetRe = regexp.MustCompile(`/([^/(]+)$`)

func parseODataPath(p string) (entitySet, keyPred string, ok bool) {
	if m := odataKeyRe.FindStringSubmatch(p); m != nil {
		return m[1], m[2], true
	}
	if m := odataSetRe.FindStringSubmatch(p); m != nil {
		return m[1], "", true
	}
	return "", "", false
}

func maskPass(p string) string {
	if len(p) <= 2 {
		return "***"
	}
	return string(p[0]) + strings.Repeat("*", len(p)-2) + string(p[len(p)-1])
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json;odata=verbose;charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeXML(w http.ResponseWriter, status int, raw string) {
	w.Header().Set("Content-Type", "application/xml;charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(xml.Header + raw))
}

func odataError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": map[string]interface{}{"lang": "en", "value": msg},
		},
	})
}

// ─── Middleware ───────────────────────────────────────────────────────────────

type server struct {
	csrf *csrfStore
	mux  *http.ServeMux
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Artificial delay
	if *delay > 0 {
		time.Sleep(time.Duration(*delay) * time.Millisecond)
	}

	// Read + log body (re-wrap for downstream handlers)
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(io.LimitReader(r.Body, 1<<20)) // max 1 MiB
		r.Body.Close()
		r.Body = io.NopCloser(strings.NewReader(string(body)))
	}
	logRequest(r, body)

	// Basic Auth (unless disabled)
	if !*noAuth {
		u, p, ok := r.BasicAuth()
		if !ok || u != *user || p != *pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="SAP NetWeaver"`)
			respHeaders := w.Header()
			logResponse(http.StatusUnauthorized, respHeaders, "BasicAuth failed")
			odataError(w, http.StatusUnauthorized, "AUTH_FAILED", "Invalid credentials")
			return
		}
	}

	// Wrap ResponseWriter to capture status
	rw := &statusWriter{ResponseWriter: w, status: 200}
	s.mux.ServeHTTP(rw, r)
	logResponse(rw.status, rw.Header(), "")
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (sw *statusWriter) WriteHeader(code int) {
	if !sw.wroteHeader {
		sw.status = code
		sw.wroteHeader = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

// ─── Route handlers ───────────────────────────────────────────────────────────

// Products: $metadata
func (s *server) handleProductMetadata(w http.ResponseWriter, r *http.Request) {
	if tok := r.Header.Get("X-Csrf-Token"); strings.EqualFold(tok, "fetch") {
		newTok := s.csrf.issue()
		w.Header().Set("X-Csrf-Token", newTok)
		w.Header().Set("Set-Cookie", "sap-usercontext=sap-client=100; path=/")
	}
	writeXML(w, http.StatusOK, productMetadata)
}

// Products: collection + single entity
func (s *server) handleProductList(w http.ResponseWriter, r *http.Request) {
	// Check for key predicate in URL: /ProductList(Product='X',Plant='Y',...)
	path := r.URL.Path
	if m := odataKeyRe.FindStringSubmatch(path); m != nil {
		s.handleProductByKey(w, r, m[2])
		return
	}
	// Collection
	top, skip := parseTopSkip(r)
	result := applyTopSkip(products, top, skip)
	writeJSON(w, http.StatusOK, v2List(result))
}

func (s *server) handleProductByKey(w http.ResponseWriter, r *http.Request, keyPred string) {
	kv := parseKeyPredicate(keyPred)
	product := stripQuotes(kv["Product"])
	plant := stripQuotes(kv["Plant"])
	valType := stripQuotes(kv["ValuationType"])

	for _, p := range products {
		if p["Product"] == product && p["Plant"] == plant && p["ValuationType"] == valType {
			writeJSON(w, http.StatusOK, v2Entity(p))
			return
		}
	}
	odataError(w, http.StatusNotFound, "RESOURCE_NOT_FOUND",
		fmt.Sprintf("Entity not found: Product='%s',Plant='%s',ValuationType='%s'", product, plant, valType))
}

// Company Code: $metadata
func (s *server) handleCompanyMetadata(w http.ResponseWriter, r *http.Request) {
	if tok := r.Header.Get("X-Csrf-Token"); strings.EqualFold(tok, "fetch") {
		newTok := s.csrf.issue()
		w.Header().Set("X-Csrf-Token", newTok)
	}
	writeXML(w, http.StatusOK, companyMetadata)
}

// Company Code: collection + single entity
func (s *server) handleCompanyCode(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if m := odataKeyRe.FindStringSubmatch(path); m != nil {
		code := stripQuotes(m[2])
		for _, cc := range companyCodes {
			if cc["CompanyCode"] == code {
				writeJSON(w, http.StatusOK, v2Entity(cc))
				return
			}
		}
		odataError(w, http.StatusNotFound, "RESOURCE_NOT_FOUND",
			fmt.Sprintf("CompanyCode '%s' not found", code))
		return
	}
	top, skip := parseTopSkip(r)
	result := applyTopSkip(companyCodes, top, skip)
	writeJSON(w, http.StatusOK, v2List(result))
}

// Sales Orders: $metadata
func (s *server) handleSalesOrderMetadata(w http.ResponseWriter, r *http.Request) {
	if tok := r.Header.Get("X-Csrf-Token"); strings.EqualFold(tok, "fetch") {
		newTok := s.csrf.issue()
		w.Header().Set("X-Csrf-Token", newTok)
	}
	writeXML(w, http.StatusOK, salesOrderMetadata)
}

// Sales Orders: collection
func (s *server) handleSalesOrders(w http.ResponseWriter, r *http.Request) {
	top, skip := parseTopSkip(r)
	result := applyTopSkip(salesOrders, top, skip)
	writeJSON(w, http.StatusOK, v2List(result))
}

// Notification: $metadata + CSRF token issuance
func (s *server) handleNotifMetadata(w http.ResponseWriter, r *http.Request) {
	if tok := r.Header.Get("X-Csrf-Token"); strings.EqualFold(tok, "fetch") {
		newTok := s.csrf.issue()
		w.Header().Set("X-Csrf-Token", newTok)
		w.Header().Set("Set-Cookie", "sap-usercontext=sap-client=100; path=/")
		fmt.Printf("   %s issued token %s\n", color(yellow, "CSRF:"), color(yellow+bold, newTok))
	}
	writeXML(w, http.StatusOK, notifMetadata)
}

// Notification: create (POST)
func (s *server) handleNotifCreate(w http.ResponseWriter, r *http.Request) {
	// Validate CSRF token (single-use)
	tok := r.Header.Get("X-Csrf-Token")
	if tok == "" {
		fmt.Printf("   %s missing X-Csrf-Token header\n", color(red, "CSRF ERR:"))
		odataError(w, http.StatusForbidden, "CSRF_TOKEN_MISSING", "X-CSRF-Token header is required")
		return
	}
	if !s.csrf.validate(tok) {
		fmt.Printf("   %s invalid/expired token: %s\n", color(red, "CSRF ERR:"), tok)
		odataError(w, http.StatusForbidden, "CSRF_TOKEN_INVALID", "CSRF token is invalid or has already been used")
		return
	}
	fmt.Printf("   %s token valid ✓\n", color(green, "CSRF:"))

	// Parse body
	var payload map[string]interface{}
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &payload); err != nil {
		odataError(w, http.StatusBadRequest, "INVALID_BODY", "Request body is not valid JSON")
		return
	}

	// Generate notification number
	reqMu.Lock()
	num := strconv.Itoa(notifCounter)
	notifCounter++
	reqMu.Unlock()

	payload["MaintenanceNotification"] = num
	writeJSON(w, http.StatusCreated, v2Entity(payload))
}

// ─── Key predicate parsing ────────────────────────────────────────────────────

// parseKeyPredicate parses "Product='3001008',Plant='1010',ValuationType='STANDARD'"
// into map[Product:3001008 Plant:1010 ValuationType:STANDARD].
func parseKeyPredicate(pred string) map[string]string {
	result := make(map[string]string)
	for _, part := range strings.Split(pred, ",") {
		idx := strings.IndexByte(part, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:idx])
		val := strings.TrimSpace(part[idx+1:])
		result[key] = val
	}
	return result
}

func stripQuotes(s string) string {
	s = strings.TrimPrefix(s, "'")
	s = strings.TrimSuffix(s, "'")
	return s
}

func parseTopSkip(r *http.Request) (top, skip int) {
	q := r.URL.Query()
	if v := q.Get("$top"); v != "" {
		top, _ = strconv.Atoi(v)
	}
	if v := q.Get("$skip"); v != "" {
		skip, _ = strconv.Atoi(v)
	}
	return
}

func applyTopSkip(data []map[string]interface{}, top, skip int) []map[string]interface{} {
	if skip > len(data) {
		return nil
	}
	data = data[skip:]
	if top > 0 && top < len(data) {
		data = data[:top]
	}
	return data
}

// ─── Router setup ─────────────────────────────────────────────────────────────

func newServer() *server {
	s := &server{
		csrf: newCSRFStore(),
		mux:  http.NewServeMux(),
	}

	// Products
	s.mux.HandleFunc("/sap/opu/odata/sap/UI_PRODUCTLIST/$metadata", s.handleProductMetadata)
	s.mux.HandleFunc("/sap/opu/odata/sap/UI_PRODUCTLIST/ProductList", s.handleProductList)
	// Catch all key-predicate variants: /ProductList(...)
	s.mux.HandleFunc("/sap/opu/odata/sap/UI_PRODUCTLIST/", s.handleProductList)

	// Company Codes
	s.mux.HandleFunc("/sap/opu/odata/sap/API_COMPANYCODE_SRV/$metadata", s.handleCompanyMetadata)
	s.mux.HandleFunc("/sap/opu/odata/sap/API_COMPANYCODE_SRV/A_CompanyCode", s.handleCompanyCode)
	s.mux.HandleFunc("/sap/opu/odata/sap/API_COMPANYCODE_SRV/", s.handleCompanyCode)

	// Sales Orders
	s.mux.HandleFunc("/sap/opu/odata/sap/SD_SALES_ORDER_IMPORT/$metadata", s.handleSalesOrderMetadata)
	s.mux.HandleFunc("/sap/opu/odata/sap/SD_SALES_ORDER_IMPORT/I_SalesOrderImport", s.handleSalesOrders)

	// Maintenance Notifications (CSRF-protected mutations)
	s.mux.HandleFunc("/sap/opu/odata/sap/API_MAINTNOTIFICATION/$metadata", s.handleNotifMetadata)
	s.mux.HandleFunc("/sap/opu/odata/sap/API_MAINTNOTIFICATION/MaintenanceNotification", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			s.handleNotifCreate(w, r)
		default:
			odataError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST is supported on this entity set")
		}
	})

	// Health + catch-all
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"UP","server":"sap-mock"}`))
	})
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		odataError(w, http.StatusNotFound, "RESOURCE_NOT_FOUND",
			fmt.Sprintf("No endpoint registered for: %s %s", r.Method, r.URL.Path))
	})

	return s
}

// ─── Banner ───────────────────────────────────────────────────────────────────

func printBanner(addr string) {
	w := os.Stdout
	sep := strings.Repeat("═", 70)
	fmt.Fprintln(w, color(cyan, sep))
	fmt.Fprintln(w, color(cyan+bold, "  SAP OData Mock Server"))
	fmt.Fprintln(w, color(gray, "  Simulates SAP NetWeaver OData v2 for local development"))
	fmt.Fprintln(w, color(cyan, sep))
	fmt.Fprintf(w, "  %-22s %s\n", color(gray, "Listen:"), color(bold, "http://"+normalizeAddr(addr)))
	fmt.Fprintf(w, "  %-22s %s\n", color(gray, "Auth:"), func() string {
		if *noAuth {
			return color(yellow, "disabled")
		}
		return color(green, "enabled")+" (user="+*user+" pass="+maskPass(*pass)+")"
	}())
	if *delay > 0 {
		fmt.Fprintf(w, "  %-22s %s ms\n", color(gray, "Artificial delay:"), strconv.Itoa(*delay))
	}
	fmt.Fprintln(w, color(cyan, sep))
	fmt.Fprintln(w)
	fmt.Fprintln(w, color(bold, "  Registered endpoints:"))
	endpoints := []string{
		"GET  /sap/opu/odata/sap/UI_PRODUCTLIST/$metadata",
		"GET  /sap/opu/odata/sap/UI_PRODUCTLIST/ProductList",
		"GET  /sap/opu/odata/sap/UI_PRODUCTLIST/ProductList(Product='X',Plant='Y',ValuationType='Z')",
		"GET  /sap/opu/odata/sap/API_COMPANYCODE_SRV/$metadata",
		"GET  /sap/opu/odata/sap/API_COMPANYCODE_SRV/A_CompanyCode",
		"GET  /sap/opu/odata/sap/API_COMPANYCODE_SRV/A_CompanyCode('code')",
		"GET  /sap/opu/odata/sap/SD_SALES_ORDER_IMPORT/$metadata",
		"GET  /sap/opu/odata/sap/SD_SALES_ORDER_IMPORT/I_SalesOrderImport",
		"GET  /sap/opu/odata/sap/API_MAINTNOTIFICATION/$metadata  [X-CSRF-Token: Fetch]",
		"POST /sap/opu/odata/sap/API_MAINTNOTIFICATION/$metadata  [X-CSRF-Token: Fetch]",
		"POST /sap/opu/odata/sap/API_MAINTNOTIFICATION/MaintenanceNotification  [CSRF required]",
		"GET  /health",
	}
	for _, e := range endpoints {
		fmt.Fprintf(w, "      %s\n", color(gray, e))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, color(gray, "  Press Ctrl+C to stop."))
	fmt.Fprintln(w)
}

func normalizeAddr(a string) string {
	if strings.HasPrefix(a, ":") {
		return "localhost" + a
	}
	return a
}

// ─── Entry point ──────────────────────────────────────────────────────────────

func main() {
	flag.Parse()

	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		*noColor = true
	}

	printBanner(*addr)

	srv := newServer()
	httpSrv := &http.Server{
		Addr:         *addr,
		Handler:      srv,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("SAP mock server listening on %s\n", *addr)
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
