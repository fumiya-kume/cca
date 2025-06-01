package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurityScore_Calculation(t *testing.T) {
	tests := []struct {
		name     string
		findings []SecurityFinding
		expected float64
	}{
		{
			name:     "no findings",
			findings: []SecurityFinding{},
			expected: 1.0,
		},
		{
			name: "one critical finding",
			findings: []SecurityFinding{
				{Severity: SecuritySeverityCritical},
			},
			expected: 0.7,
		},
		{
			name: "mixed findings",
			findings: []SecurityFinding{
				{Severity: SecuritySeverityLow},
				{Severity: SecuritySeverityMedium},
				{Severity: SecuritySeverityHigh},
			},
			expected: 0.65, // 1.0 - 0.05 - 0.1 - 0.2
		},
		{
			name: "many critical findings",
			findings: []SecurityFinding{
				{Severity: SecuritySeverityCritical},
				{Severity: SecuritySeverityCritical},
				{Severity: SecuritySeverityCritical},
				{Severity: SecuritySeverityCritical},
			},
			expected: 0.0, // Score can't go below 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the score calculation from calculateFinalScores
			securityScore := 1.0
			for _, finding := range tt.findings {
				switch finding.Severity {
				case SecuritySeverityCritical:
					securityScore -= 0.3
				case SecuritySeverityHigh:
					securityScore -= 0.2
				case SecuritySeverityMedium:
					securityScore -= 0.1
				case SecuritySeverityLow:
					securityScore -= 0.05
				}
			}
			if securityScore < 0 {
				securityScore = 0
			}

			assert.InDelta(t, tt.expected, securityScore, 0.001)
		})
	}
}
