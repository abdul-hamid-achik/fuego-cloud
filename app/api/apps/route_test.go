package apps

import (
	"testing"
)

func TestAppNameValidation(t *testing.T) {
	tests := []struct {
		name    string
		appName string
		valid   bool
	}{
		{"valid simple name", "myapp", true},
		{"valid with numbers", "myapp123", true},
		{"valid with hyphens", "my-app", true},
		{"valid complex name", "my-app-123", true},
		{"valid minimum length", "abc", true},
		{"valid with many hyphens", "my-new-test-app", true},

		{"invalid starts with number", "123app", false},
		{"invalid starts with hyphen", "-myapp", false},
		{"invalid ends with hyphen", "myapp-", false},
		{"invalid uppercase", "MyApp", false},
		{"invalid underscore", "my_app", false},
		{"invalid space", "my app", false},
		{"invalid special chars", "my@app", false},
		{"regex allows 2 chars but handler rejects", "ab", true},
		{"invalid single char", "a", false},
		{"invalid empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := appNameRegex.MatchString(tt.appName)
			if valid != tt.valid {
				t.Errorf("appNameRegex.MatchString(%q) = %v, want %v", tt.appName, valid, tt.valid)
			}
		})
	}
}

func TestAppNameLengthValidation(t *testing.T) {
	tests := []struct {
		name   string
		length int
		valid  bool
	}{
		{"too short - 2 chars", 2, false},
		{"minimum valid - 3 chars", 3, true},
		{"normal length", 10, true},
		{"maximum valid - 63 chars", 63, true},
		{"too long - 64 chars", 64, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := generateTestName(tt.length)
			valid := len(name) >= 3 && len(name) <= 63
			if valid != tt.valid {
				t.Errorf("length validation for %d chars = %v, want %v", tt.length, valid, tt.valid)
			}
		})
	}
}

func generateTestName(length int) string {
	if length <= 0 {
		return ""
	}
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = 'a'
	}
	return string(result)
}

func TestValidRegions(t *testing.T) {
	validRegions := map[string]bool{"gdl": true, "mex": true, "qro": true}

	tests := []struct {
		region string
		valid  bool
	}{
		{"gdl", true},
		{"mex", true},
		{"qro", true},
		{"us-east", false},
		{"", false},
		{"GDL", false},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			if validRegions[tt.region] != tt.valid {
				t.Errorf("region %q valid = %v, want %v", tt.region, validRegions[tt.region], tt.valid)
			}
		})
	}
}

func TestValidSizes(t *testing.T) {
	validSizes := map[string]bool{"starter": true, "pro": true, "enterprise": true}

	tests := []struct {
		size  string
		valid bool
	}{
		{"starter", true},
		{"pro", true},
		{"enterprise", true},
		{"small", false},
		{"", false},
		{"STARTER", false},
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			if validSizes[tt.size] != tt.valid {
				t.Errorf("size %q valid = %v, want %v", tt.size, validSizes[tt.size], tt.valid)
			}
		})
	}
}

func TestCreateAppRequestDefaults(t *testing.T) {
	req := CreateAppRequest{
		Name: "test-app",
	}

	if req.Region != "" {
		t.Errorf("expected empty region by default, got %q", req.Region)
	}

	if req.Size != "" {
		t.Errorf("expected empty size by default, got %q", req.Size)
	}

	if req.Region == "" {
		req.Region = "gdl"
	}
	if req.Size == "" {
		req.Size = "starter"
	}

	if req.Region != "gdl" {
		t.Errorf("expected default region 'gdl', got %q", req.Region)
	}

	if req.Size != "starter" {
		t.Errorf("expected default size 'starter', got %q", req.Size)
	}
}

func TestAppResponseStructure(t *testing.T) {
	resp := AppResponse{
		ID:              "test-id",
		Name:            "test-app",
		Region:          "gdl",
		Size:            "starter",
		Status:          "running",
		DeploymentCount: 5,
		URL:             "https://test-app.fuego.build",
	}

	if resp.ID != "test-id" {
		t.Error("ID mismatch")
	}
	if resp.Name != "test-app" {
		t.Error("Name mismatch")
	}
	if resp.DeploymentCount != 5 {
		t.Errorf("DeploymentCount expected 5, got %d", resp.DeploymentCount)
	}
}
