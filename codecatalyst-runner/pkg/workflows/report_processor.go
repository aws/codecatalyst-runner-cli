package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/owenrumney/go-sarif/sarif"
	"github.com/rs/zerolog/log"
)

type reportHandler func(reader io.Reader, report *Report) error

// ReportProcessor looks for reports created by the action and fails if they dont meet the successCriteria.
// Results are saved in the provided report parameter.
func ReportProcessor(
	report *Report,
	successCriteria *SuccessCriteria,
	reportDir string,
) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER ReportProcessor")
		err := e(ctx)
		if processError := newReportProcessor(
			reportDir,
			report,
			successCriteria,
		)(ctx); processError != nil {
			log.Warn().Err(processError).Msg("Failed to process report")
		}
		if report.Result != ResultSucceeded {
			err = fmt.Errorf("report status %s", report.Result)
		}
		log.Ctx(ctx).Debug().Msg("EXIT ReportProcessor")
		return err
	}
}

func newReportProcessor(reportsDir string, report *Report, successCriteria *SuccessCriteria) common.Executor {
	handlers := []reportHandler{
		sarifReportHandler(successCriteria.VulnerabilityThreshold),
	}
	return func(ctx context.Context) error {
		err := filepath.WalkDir(reportsDir, func(path string, d fs.DirEntry, err error) error {
			if d != nil && d.Type().IsRegular() {
				for _, handler := range handlers {
					reportFile, err := os.Open(path)
					if err != nil {
						return err
					}
					defer reportFile.Close()
					if err := handler(reportFile, report); err != nil {
						return nil
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		if report.Result == "" {
			report.Result = ResultSucceeded
		}
		return nil
	}
}

func sarifReportHandler(severityThreshold VulnerabilitySeverity) reportHandler {
	return func(reader io.Reader, report *Report) error {
		decoder := json.NewDecoder(reader)
		sarifReport := new(sarif.Report)
		if err := decoder.Decode(sarifReport); err != nil {
			log.Debug().Err(err).Msgf("Skipping non-sarif report")
			return nil
		}
		if strings.HasPrefix(path.Base(sarifReport.Schema), "sarif") {
			for _, run := range sarifReport.Runs {
				for _, r := range run.Results {
					// only consider results with empty 'kind' or 'kind' of 'fail'
					if r.Kind == nil || *r.Kind == "" || *r.Kind == "fail" {
						severity := levelToSeverity(r.Level)
						log.Debug().Msgf("Got result with severity %s (threshold=%s)", severity, severityThreshold)
						if severityExceedsThreshold(severityThreshold, severity) && len(r.Suppressions) == 0 {
							report.Result = ResultFailed
						}
						report.Vulnerabilities = append(report.Vulnerabilities, Vulnerability{
							Severity:     severity,
							RuleID:       safeString(r.RuleID),
							Message:      safeString(r.Message.Text),
							Locations:    convertLocations(r.Locations),
							Suppressions: convertSuppressions(r.Suppressions),
						})
					}
				}
			}
		}
		return nil
	}
}

func severityExceedsThreshold(severityThreshold VulnerabilitySeverity, severity VulnerabilitySeverity) bool {
	return severityOrdinal(severity) >= severityOrdinal(severityThreshold)
}

func severityOrdinal(severity VulnerabilitySeverity) int {
	switch severity {
	case VulnerabilitySeverityCritical:
		return 1000
	case VulnerabilitySeverityHigh:
		return 500
	case VulnerabilitySeverityMedium:
		return 100
	case VulnerabilitySeverityLow:
		return 10
	case VulnerabilitySeverityInformational:
		return 1
	default:
		return 0
	}
}

func convertLocations(sarifLocations []*sarif.Location) []Location {
	if sarifLocations == nil {
		return nil
	}
	locations := make([]Location, 0)
	for _, l := range sarifLocations {
		if l == nil {
			continue
		}

		location := Location{}
		if l.PhysicalLocation != nil {
			if l.PhysicalLocation.ArtifactLocation != nil {
				location.URI = safeString(l.PhysicalLocation.ArtifactLocation.URI)
			}
			if l.PhysicalLocation.Region != nil {
				location.StartLine = l.PhysicalLocation.Region.StartLine
				location.EndLine = l.PhysicalLocation.Region.EndLine
				if l.PhysicalLocation.Region.Snippet != nil {
					location.Snippet = safeString(l.PhysicalLocation.Region.Snippet.Text)
				}
			}
		}
		locations = append(locations, location)
	}
	return locations
}

func convertSuppressions(sarifSuppressions []*sarif.Suppression) []Suppression {
	if sarifSuppressions == nil {
		return nil
	}
	suppressions := make([]Suppression, 0)
	for _, s := range sarifSuppressions {
		if s == nil {
			continue
		}

		suppression := Suppression{
			Kind:          s.Kind,
			Justification: safeString(s.Justification),
		}
		suppressions = append(suppressions, suppression)
	}
	return suppressions
}

func levelToSeverity(level *string) VulnerabilitySeverity {
	if level == nil {
		return VulnerabilitySeverityMedium
	}
	switch *level {
	case "error":
		return VulnerabilitySeverityHigh
	case "warning":
		return VulnerabilitySeverityMedium
	case "note":
		return VulnerabilitySeverityLow
	case "none":
		return VulnerabilitySeverityInformational
	default:
		return VulnerabilitySeverityMedium
	}
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Report object is the aggregation of all reports detected in the action
type Report struct {
	Result          Result          `json:"codecatalyst_action_result"`                   // result of the report
	PassRate        *float32        `json:"codecatalyst_action_passRate,omitempty"`       // number between 0 and 100 representing the percentage of tests that passed
	LineCoverage    *float32        `json:"codecatalyst_action_lineCoverage,omitempty"`   // number between 0 and 100 representing the percentage of lines that were covered by tests
	BranchCoverage  *float32        `json:"codecatalyst_action_branchCoverage,omitempty"` // number between 0 and 100 representing the percentage of branches that were covered by tests
	Vulnerabilities []Vulnerability `json:"codecatalyst_action_vulnerabilities"`          // list of vulnerabilities found
}

// Result for a report, either SUCCEEDED or FAILED
type Result string

const (
	// ResultSucceeded indicates that the action passed
	ResultSucceeded Result = "SUCCEEDED"
	// ResultFailed indicates that the action failed
	ResultFailed Result = "FAILED"
)

// Vulnerability found during an execution of an action
type Vulnerability struct {
	Severity     VulnerabilitySeverity // severity of the vulnerability
	RuleID       string                // ID of the rule that found the vulnerability
	Message      string                // description of the vulnerability
	Locations    []Location            // locations of the vulnerability
	Suppressions []Suppression         // list of suppressions applied to the vulnerability
}

// Location of a vulnerability
type Location struct {
	URI       string // uri of the location
	StartLine *int   `json:",omitempty"` // first line number of a location
	EndLine   *int   `json:",omitempty"` // last line number of a location
	Snippet   string // portion of the artifact identified in the location
}

// Suppression object describes a request to suppress a result
type Suppression struct {
	Kind          string // type of suppression, one of: inSource or external
	Justification string // user-supplied string explaining why the result was suppressed
}

// SuccessCriteria defines the required results of test reports for an action to pass
type SuccessCriteria struct {
	PassRate               float32               `yaml:"passRate"`               // number between 0 and 100 representing the percentage of tests that must pass
	LineCoverage           float32               `yaml:"lineCoverage"`           // number between 0 and 100 representing the percentage of lines that must be covered by tests
	BranchCoverage         float32               `yaml:"branchCoverage"`         // number between 0 and 100 representing the percentage of branches that must be covered by tests
	VulnerabilityThreshold VulnerabilitySeverity `yaml:"vulnerabilityThreshold"` // the max severity of the vulnerabilities allowed
}

// VulnerabilitySeverity describes the severity of a vulnerability
type VulnerabilitySeverity string

const (
	// VulnerabilitySeverityCritical is critical severity
	VulnerabilitySeverityCritical VulnerabilitySeverity = "CRITICAL"
	// VulnerabilitySeverityHigh is high severity
	VulnerabilitySeverityHigh VulnerabilitySeverity = "HIGH"
	// VulnerabilitySeverityMedium is medium severity
	VulnerabilitySeverityMedium VulnerabilitySeverity = "MEDIUM"
	// VulnerabilitySeverityLow is low severity
	VulnerabilitySeverityLow VulnerabilitySeverity = "LOW"
	// VulnerabilitySeverityInformational is informational severity
	VulnerabilitySeverityInformational VulnerabilitySeverity = "INFORMATIONAL"
)
