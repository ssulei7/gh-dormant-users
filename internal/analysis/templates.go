package analysis

// AnalysisTemplate represents a predefined analysis template
type AnalysisTemplate struct {
	Name        string
	Description string
	Prompt      string
}

// PredefinedTemplates contains all available analysis templates
var PredefinedTemplates = map[string]AnalysisTemplate{
	"summary": {
		Name:        "Summary Analysis",
		Description: "Generate a high-level summary of dormant user data",
		Prompt: `You are analyzing a GitHub organization's user activity report. Based on the pre-aggregated statistics below, provide a clear executive summary including:

1. Overall health assessment of user engagement
2. Key metrics interpretation (active vs dormant ratio)
3. Notable patterns in activity types
4. Brief recommendations for next steps

Keep the response concise and actionable.

%s`,
	},
	"trends": {
		Name:        "Activity Trends",
		Description: "Identify patterns and trends in user activity",
		Prompt: `You are analyzing a GitHub organization's user activity report. Based on the statistics below, identify trends and patterns:

1. Which activity types dominate among active users and why this matters
2. What the activity distribution suggests about team workflows
3. Potential correlations between activity types
4. Specific recommendations to increase engagement based on these patterns

%s`,
	},
	"risk": {
		Name:        "Risk Assessment",
		Description: "Assess risks associated with dormant accounts",
		Prompt: `You are a security analyst reviewing a GitHub organization's user activity report. Based on the statistics below, provide a risk assessment:

1. Security risk level (low/medium/high) based on dormant account percentage
2. Specific security concerns with dormant accounts (credential exposure, unused permissions)
3. Compliance implications (license optimization, access review requirements)
4. Prioritized remediation steps with timeline recommendations

%s`,
	},
	"recommendations": {
		Name:        "Action Recommendations",
		Description: "Get actionable recommendations for managing dormant users",
		Prompt: `You are an IT administrator planning user lifecycle management. Based on the statistics below, create an action plan:

1. Immediate actions (within 1 week)
2. Short-term actions (within 1 month)
3. Policy recommendations for preventing future dormancy
4. Communication templates for reaching out to dormant users
5. Criteria for account deactivation decisions

Be specific and provide actionable steps.

%s`,
	},
	"custom": {
		Name:        "Custom Analysis",
		Description: "Perform a custom analysis with user-provided prompt",
		Prompt:      "%s\n\n%s",
	},
}

// GetTemplateNames returns all available template names
func GetTemplateNames() []string {
	names := make([]string, 0, len(PredefinedTemplates))
	for name := range PredefinedTemplates {
		names = append(names, name)
	}
	return names
}

// GetTemplate returns a template by name, or nil if not found
func GetTemplate(name string) *AnalysisTemplate {
	if template, ok := PredefinedTemplates[name]; ok {
		return &template
	}
	return nil
}

// GetTemplateDescriptions returns a map of template names to descriptions
func GetTemplateDescriptions() map[string]string {
	descriptions := make(map[string]string)
	for name, template := range PredefinedTemplates {
		descriptions[name] = template.Description
	}
	return descriptions
}
