package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileCreate(t *testing.T) {
	// Use temporary directory for testing
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		// Windows
		originalHome = os.Getenv("USERPROFILE")
	}

	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("USERPROFILE", tempDir)

	tests := []struct {
		name          string
		profileName   string
		url           string
		user          string
		pass          string
		token         string
		expectError   bool
		errorContains string
	}{
		{
			name:        "create valid profile",
			profileName: "test-profile",
			url:         "https://api.example.com/odata",
			user:        "testuser",
			pass:        "testpass",
			expectError: false,
		},
		{
			name:          "missing name",
			profileName:   "",
			url:           "https://api.example.com/odata",
			expectError:   true,
			errorContains: "name is required",
		},
		{
			name:          "missing URL",
			profileName:   "test",
			url:           "",
			expectError:   true,
			errorContains: "URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := profileCreateCommand(tt.profileName, tt.url, tt.user, tt.pass, tt.token)

			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectError && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error '%v' does not contain '%s'", err, tt.errorContains)
				}
			}
		})
	}
}

func TestGetConnection(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		user        string
		pass        string
		token       string
		profile     string
		expectError bool
	}{
		{
			name:        "connection with URL",
			url:         "https://api.example.com/odata",
			expectError: false,
		},
		{
			name:          "missing URL",
			url:           "",
			profile:       "",
			expectError:   true,
		},
		{
			name:        "connection with auth",
			url:         "https://api.example.com/odata",
			user:        "testuser",
			pass:        "testpass",
			expectError: false,
		},
		{
			name:        "connection with token",
			url:         "https://api.example.com/odata",
			token:       "test-token",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := getConnection(tt.url, tt.user, tt.pass, tt.token, tt.profile, 30)

			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectError && conn != nil {
				if conn.URL != tt.url {
					t.Errorf("URL mismatch: expected %s, got %s", tt.url, conn.URL)
				}
				if conn.Username != tt.user {
					t.Errorf("Username mismatch: expected %s, got %s", tt.user, conn.Username)
				}
				if conn.Token != tt.token {
					t.Errorf("Token mismatch: expected %s, got %s", tt.token, conn.Token)
				}
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{nil, "null"},
		{"hello", "hello"},
		{42, "42"},
		{3.14, "3.14"},
		{true, "true"},
		{false, "false"},
		{float64(100), "100"},
		{float64(100.5), "100.5"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			result := toString(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatOutput(t *testing.T) {
	data := []map[string]interface{}{
		{
			"Name":  "Product A",
			"Price": 100.50,
			"Stock": 10,
		},
		{
			"Name":  "Product B",
			"Price": 200.00,
			"Stock": 5,
		},
	}

	tests := []struct {
		name        string
		format      string
		shouldError bool
	}{
		{
			name:        "JSON format",
			format:      "json",
			shouldError: false,
		},
		{
			name:        "Table format",
			format:      "table",
			shouldError: false,
		},
		{
			name:        "Text format",
			format:      "text",
			shouldError: false,
		},
		{
			name:        "Default format",
			format:      "",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatOutput(data, tt.format)

			if tt.shouldError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestExportToCSV(t *testing.T) {
	data := []map[string]interface{}{
		{
			"Name":  "Product A",
			"Price": 100.50,
		},
		{
			"Name":  "Product B",
			"Price": 200.00,
		},
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	err = exportToCSV(tmpFile, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify file was written
	stat, err := tmpFile.Stat()
	if err != nil {
		t.Errorf("failed to stat file: %v", err)
	}
	if stat.Size() == 0 {
		t.Errorf("CSV file is empty")
	}
}

func TestQueryOptions(t *testing.T) {
	opts := QueryOptions{
		Filter:  "Price gt 100",
		Select:  "Name,Price",
		OrderBy: "Name asc",
		Skip:    0,
		Top:     10,
	}

	// Just verify the struct can be created and accessed
	if opts.Filter != "Price gt 100" {
		t.Errorf("Filter not set correctly")
	}
	if opts.Top != 10 {
		t.Errorf("Top not set correctly")
	}
}

func TestExportOptions(t *testing.T) {
	opts := ExportOptions{
		Format: "json",
		Output: "export.json",
		Filter: "Status eq 'Active'",
		Select: "Name,Price",
		Limit:  1000,
	}

	// Just verify the struct can be created and accessed
	if opts.Format != "json" {
		t.Errorf("Format not set correctly")
	}
	if opts.Output != "export.json" {
		t.Errorf("Output not set correctly")
	}
	if opts.Limit != 1000 {
		t.Errorf("Limit not set correctly")
	}
}

func TestProfileDelete(t *testing.T) {
	// Use temporary directory for testing
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("USERPROFILE", tempDir)

	// First create a profile
	err := profileCreateCommand("test-delete", "https://api.example.com/odata", "", "", "")
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	// Then delete it
	err = profileDeleteCommand("test-delete")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify it's deleted
	_, err = loadProfile("test-delete")
	if err == nil {
		t.Errorf("expected error when loading deleted profile")
	}
}

func TestConnection(t *testing.T) {
	conn := &Connection{
		URL:      "https://api.example.com/odata",
		Username: "user",
		Password: "pass",
		Token:    "",
		Timeout:  30,
	}

	if conn.URL != "https://api.example.com/odata" {
		t.Errorf("URL not set correctly")
	}
	if conn.Username != "user" {
		t.Errorf("Username not set correctly")
	}
	if conn.Timeout != 30 {
		t.Errorf("Timeout not set correctly")
	}
}

func TestEmptyDataFormatTable(t *testing.T) {
	data := []map[string]interface{}{}

	err := formatTable(data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSingleRowFormatTable(t *testing.T) {
	data := []map[string]interface{}{
		{
			"Name": "Product A",
			"ID":   1,
		},
	}

	err := formatTable(data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestJSONMarshal(t *testing.T) {
	data := []map[string]interface{}{
		{
			"Name":  "Test",
			"Value": 123,
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Errorf("failed to marshal JSON: %v", err)
	}

	if len(jsonData) == 0 {
		t.Errorf("JSON output is empty")
	}
}

func TestProfilesPathCreation(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("USERPROFILE", tempDir)

	path, err := getProfilesPath()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.Contains(path, ".traverse") {
		t.Errorf("path should contain .traverse directory")
	}

	// Verify directory was created
	dir := filepath.Dir(path)
	stat, err := os.Stat(dir)
	if err != nil {
		t.Errorf("directory not created: %v", err)
	}
	if !stat.IsDir() {
		t.Errorf("path is not a directory")
	}
}

func TestQueryOptionsBuilding(t *testing.T) {
	opts := QueryOptions{
		Filter:  "Name eq 'Test'",
		Select:  "ID,Name",
		OrderBy: "Name desc",
		Skip:    5,
		Top:     20,
	}

	if opts.Skip != 5 {
		t.Errorf("Skip not set correctly")
	}
	if opts.Filter != "Name eq 'Test'" {
		t.Errorf("Filter not set correctly")
	}
}

func TestLoadProfilesNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("USERPROFILE", tempDir)

	cfg, err := loadProfiles()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if cfg == nil {
		t.Errorf("expected config, got nil")
	}

	if len(cfg.Profiles) != 0 {
		t.Errorf("expected empty profiles, got %d", len(cfg.Profiles))
	}
}
