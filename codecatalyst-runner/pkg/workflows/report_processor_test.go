package workflows

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestReportFeature(t *testing.T) {
	type TestParams struct {
		TestCase                string
		ReportDir               string
		SuccessCriteria         SuccessCriteria
		ExpectedVulnerabilities []Vulnerability
		ExpectedResult          Result
	}

	for _, tt := range []*TestParams{
		{
			TestCase:                "Empty report",
			ReportDir:               "testdata/reports/empty",
			SuccessCriteria:         SuccessCriteria{VulnerabilityThreshold: VulnerabilitySeverityCritical},
			ExpectedVulnerabilities: nil,
			ExpectedResult:          ResultSucceeded,
		},
		{
			TestCase:                "Non-SARIF report",
			ReportDir:               "testdata/reports/non-sarif",
			SuccessCriteria:         SuccessCriteria{VulnerabilityThreshold: VulnerabilitySeverityCritical},
			ExpectedVulnerabilities: nil,
			ExpectedResult:          ResultSucceeded,
		},
		{
			TestCase:                "Simple report - no findings",
			ReportDir:               "testdata/reports/sarif-no-findings",
			SuccessCriteria:         SuccessCriteria{VulnerabilityThreshold: VulnerabilitySeverityCritical},
			ExpectedVulnerabilities: nil,
			ExpectedResult:          ResultSucceeded,
		},
		{
			TestCase:        "Simple report - below threshold",
			ReportDir:       "testdata/reports/sarif-low-severity",
			SuccessCriteria: SuccessCriteria{VulnerabilityThreshold: VulnerabilitySeverityCritical},
			ExpectedResult:  ResultSucceeded,
			ExpectedVulnerabilities: []Vulnerability{
				{
					Severity: VulnerabilitySeverityMedium,
					RuleID:   "no-unused-vars",
					Message:  "'x' is assigned a value but never used.",
					Locations: []Location{
						{
							URI:       "file:///C:/dev/sarif/sarif-tutorials/samples/Introduction/simple-example.js",
							StartLine: intRef(1),
						},
					},
				},
			},
		},
		{
			TestCase:        "Simple report - with vulnerabilities",
			ReportDir:       "testdata/reports/sarif-high-severity",
			SuccessCriteria: SuccessCriteria{VulnerabilityThreshold: VulnerabilitySeverityHigh},
			ExpectedResult:  ResultFailed,
			ExpectedVulnerabilities: []Vulnerability{
				{
					Severity: VulnerabilitySeverityHigh,
					RuleID:   "no-unused-vars",
					Message:  "'x' is assigned a value but never used.",
					Locations: []Location{
						{
							URI:       "file:///C:/dev/sarif/sarif-tutorials/samples/Introduction/simple-example.js",
							StartLine: intRef(1),
							EndLine:   intRef(3),
							Snippet:   "var x;",
						},
					},
				},
			},
		},
	} {
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			// setup the code under test
			ctx := context.Background()
			report := new(Report)
			feature := ReportProcessor(report, &tt.SuccessCriteria, tt.ReportDir)

			// setup the mock
			m := new(runner.MockPlanExecutor)
			m.OnExecute(ctx).Return(nil)

			// run the feature
			err := m.Execute(ctx, feature)

			// assert the results
			if tt.ExpectedResult == ResultSucceeded {
				assert.NoError(err, "%s - err", tt.TestCase)
			} else {
				assert.Error(err, "%s - err", tt.TestCase)
			}
			m.AssertExpectations(t)

			assert.Equal(tt.ExpectedResult, report.Result, "%s - result", tt.TestCase)
			assert.Equal(tt.ExpectedVulnerabilities, report.Vulnerabilities, "%s - vulnerabilities", tt.TestCase)
		})
	}
}

func TestSarifReportHandler(t *testing.T) {
	assert := assert.New(t)

	type TestParams struct {
		TestCase                string
		Report                  string
		SuccessCriteria         SuccessCriteria
		ExpectedVulnerabilities []Vulnerability
		ExpectedResult          Result
	}

	for _, tt := range []*TestParams{
		{
			TestCase:                "Empty report",
			Report:                  "",
			SuccessCriteria:         SuccessCriteria{VulnerabilityThreshold: VulnerabilitySeverityCritical},
			ExpectedVulnerabilities: nil,
		},
		{
			TestCase:                "Non-SARIF report",
			Report:                  "<test>hello world</test>",
			SuccessCriteria:         SuccessCriteria{VulnerabilityThreshold: VulnerabilitySeverityCritical},
			ExpectedVulnerabilities: nil,
		},
		{
			TestCase:                "Simple report - no findings",
			Report:                  SarifEmpty,
			SuccessCriteria:         SuccessCriteria{VulnerabilityThreshold: VulnerabilitySeverityCritical},
			ExpectedVulnerabilities: nil,
		},
		{
			TestCase:        "Simple report - below threshold",
			Report:          SarifWithWarning,
			SuccessCriteria: SuccessCriteria{VulnerabilityThreshold: VulnerabilitySeverityCritical},
			ExpectedVulnerabilities: []Vulnerability{
				{
					Severity: VulnerabilitySeverityMedium,
					RuleID:   "no-unused-vars",
					Message:  "'x' is assigned a value but never used.",
					Locations: []Location{
						{
							URI:       "file:///C:/dev/sarif/sarif-tutorials/samples/Introduction/simple-example.js",
							StartLine: intRef(1),
						},
					},
				},
			},
		},
		{
			TestCase:        "Simple report - with vulnerabilities",
			Report:          SarifWithError,
			SuccessCriteria: SuccessCriteria{VulnerabilityThreshold: VulnerabilitySeverityHigh},
			ExpectedResult:  ResultFailed,
			ExpectedVulnerabilities: []Vulnerability{
				{
					Severity: VulnerabilitySeverityHigh,
					RuleID:   "no-unused-vars",
					Message:  "'x' is assigned a value but never used.",
					Locations: []Location{
						{
							URI:       "file:///C:/dev/sarif/sarif-tutorials/samples/Introduction/simple-example.js",
							StartLine: intRef(1),
							EndLine:   intRef(3),
							Snippet:   "var x;",
						},
					},
				},
			},
		},
	} {
		report := new(Report)
		err := sarifReportHandler(tt.SuccessCriteria.VulnerabilityThreshold)(strings.NewReader(tt.Report), report)
		assert.Equal(tt.ExpectedResult, report.Result, "%s - result", tt.TestCase)
		assert.Equal(tt.ExpectedVulnerabilities, report.Vulnerabilities, "%s - vulnerabilities", tt.TestCase)
		assert.NoError(err)
	}
}

func intRef(i int) *int {
	return &i
}

const SarifEmpty = `
{
  "version": "2.1.0",
  "$schema": "http://json.schemastore.org/sarif-2.1.0-rtm.4",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "ESLint",
          "informationUri": "https://eslint.org",
          "rules": [
            {
              "id": "no-unused-vars",
              "shortDescription": {
                "text": "disallow unused variables"
              },
              "helpUri": "https://eslint.org/docs/rules/no-unused-vars",
              "properties": {
                "category": "Variables"
              }
            }
          ]
        }
      },
      "artifacts": [
        {
          "location": {
            "uri": "file:///C:/dev/sarif/sarif-tutorials/samples/Introduction/simple-example.js"
          }
        }
      ],
      "results": [
      ]
    }
  ]
}
`
const SarifWithWarning = `
{
  "version": "2.1.0",
  "$schema": "http://json.schemastore.org/sarif-2.1.0-rtm.4",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "ESLint",
          "informationUri": "https://eslint.org",
          "rules": [
            {
              "id": "no-unused-vars",
              "shortDescription": {
                "text": "disallow unused variables"
              },
              "helpUri": "https://eslint.org/docs/rules/no-unused-vars",
              "properties": {
                "category": "Variables"
              }
            }
          ]
        }
      },
      "artifacts": [
        {
          "location": {
            "uri": "file:///C:/dev/sarif/sarif-tutorials/samples/Introduction/simple-example.js"
          }
        }
      ],
      "results": [
        {
          "message": {
            "text": "'x' is assigned a value but never used."
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "file:///C:/dev/sarif/sarif-tutorials/samples/Introduction/simple-example.js",
                  "index": 0
                },
                "region": {
                  "startLine": 1,
                  "startColumn": 5
                }
              }
            }
          ],
          "ruleId": "no-unused-vars",
          "ruleIndex": 0
        }
      ]
    }
  ]
}
`
const SarifWithError = `
{
  "version": "2.1.0",
  "$schema": "http://json.schemastore.org/sarif-2.1.0-rtm.4",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "ESLint",
          "informationUri": "https://eslint.org",
          "rules": [
            {
              "id": "no-unused-vars",
              "shortDescription": {
                "text": "disallow unused variables"
              },
              "helpUri": "https://eslint.org/docs/rules/no-unused-vars",
              "properties": {
                "category": "Variables"
              }
            }
          ]
        }
      },
      "artifacts": [
        {
          "location": {
            "uri": "file:///C:/dev/sarif/sarif-tutorials/samples/Introduction/simple-example.js"
          }
        }
      ],
      "results": [
        {
          "level": "error",
          "message": {
            "text": "'x' is assigned a value but never used."
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "file:///C:/dev/sarif/sarif-tutorials/samples/Introduction/simple-example.js",
                  "index": 0
                },
                "region": {
                  "startLine": 1,
                  "startColumn": 5,
									"endLine": 3,
									"snippet": {
										"text": "var x;"
									}
                }
              }
            }
          ],
          "ruleId": "no-unused-vars",
          "ruleIndex": 0
        }
      ]
    }
  ]
}
`
