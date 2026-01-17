package analysis

import (
	"testing"
)

func TestGetTemplateNames(t *testing.T) {
	names := GetTemplateNames()

	if len(names) == 0 {
		t.Error("GetTemplateNames() returned empty slice")
	}

	// Check that all expected templates are present
	expectedTemplates := []string{"summary", "trends", "risk", "recommendations", "custom"}
	for _, expected := range expectedTemplates {
		found := false
		for _, name := range names {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected template %q not found in GetTemplateNames()", expected)
		}
	}
}

func TestGetTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		expectNil    bool
	}{
		{"valid summary template", "summary", false},
		{"valid trends template", "trends", false},
		{"valid risk template", "risk", false},
		{"valid recommendations template", "recommendations", false},
		{"valid custom template", "custom", false},
		{"invalid template", "nonexistent", true},
		{"empty template name", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTemplate(tt.templateName)
			if tt.expectNil && result != nil {
				t.Errorf("GetTemplate(%q) = %v, want nil", tt.templateName, result)
			}
			if !tt.expectNil && result == nil {
				t.Errorf("GetTemplate(%q) = nil, want non-nil", tt.templateName)
			}
		})
	}
}

func TestGetTemplateDescriptions(t *testing.T) {
	descriptions := GetTemplateDescriptions()

	if len(descriptions) == 0 {
		t.Error("GetTemplateDescriptions() returned empty map")
	}

	// Check that all predefined templates have descriptions
	for name := range PredefinedTemplates {
		if _, ok := descriptions[name]; !ok {
			t.Errorf("Template %q missing from GetTemplateDescriptions()", name)
		}
	}

	// Verify descriptions are not empty
	for name, desc := range descriptions {
		if desc == "" {
			t.Errorf("Template %q has empty description", name)
		}
	}
}

func TestPredefinedTemplatesStructure(t *testing.T) {
	for name, template := range PredefinedTemplates {
		t.Run(name, func(t *testing.T) {
			if template.Name == "" {
				t.Errorf("Template %q has empty Name field", name)
			}
			if template.Description == "" {
				t.Errorf("Template %q has empty Description field", name)
			}
			if template.Prompt == "" {
				t.Errorf("Template %q has empty Prompt field", name)
			}
		})
	}
}
