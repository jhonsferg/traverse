package sap

import (
	"testing"
)

// TestURLConstructionEdgeCases verifies proper URL normalization
// without multiple slashes or other construction issues (BUG-006 fix).
func TestURLConstructionEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		systemURL      string
		client         string
		service        string
		expectedBase   string
		expectedClient string
	}{
		{
			name:         "Normal case without trailing slash",
			systemURL:    "https://s4h.example.com:44300",
			client:       "100",
			service:      "MM_MATERIAL_SRV",
			expectedBase: "https://s4h.example.com:44300/sap/opu/odata/sap/MM_MATERIAL_SRV?sap-client=100",
		},
		{
			name:         "SystemURL with trailing slash",
			systemURL:    "https://s4h.example.com:44300/",
			client:       "100",
			service:      "MM_MATERIAL_SRV",
			expectedBase: "https://s4h.example.com:44300/sap/opu/odata/sap/MM_MATERIAL_SRV?sap-client=100",
		},
		{
			name:         "No client number",
			systemURL:    "https://s4h.example.com:44300",
			client:       "",
			service:      "MM_MATERIAL_SRV",
			expectedBase: "https://s4h.example.com:44300/sap/opu/odata/sap/MM_MATERIAL_SRV",
		},
		{
			name:         "Special characters in service name",
			systemURL:    "https://s4h.example.com:44300",
			client:       "200",
			service:      "C4C_API_SRV_V2",
			expectedBase: "https://s4h.example.com:44300/sap/opu/odata/sap/C4C_API_SRV_V2?sap-client=200",
		},
		{
			name:         "URL encoding test",
			systemURL:    "https://s4h.example.com:44300",
			client:       "100-DEV",
			service:      "MM_MATERIAL_SRV",
			expectedBase: "https://s4h.example.com:44300/sap/opu/odata/sap/MM_MATERIAL_SRV?sap-client=100-DEV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &sapConfig{}
			opt := WithSAPBaseURL(tt.systemURL, tt.client, tt.service)
			err := opt(cfg)
			if err != nil {
				t.Fatalf("WithSAPBaseURL failed: %v", err)
			}

			if cfg.baseURL != tt.expectedBase {
				t.Errorf("URL mismatch:\n  Expected: %s\n  Got:      %s", tt.expectedBase, cfg.baseURL)
			}

			// Verify no multiple slashes (BUG-006 check)
			if contains(cfg.baseURL, "///") {
				t.Errorf("URL contains multiple slashes (///): %s", cfg.baseURL)
			}
		})
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
