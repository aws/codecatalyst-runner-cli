package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"
)

// ActionOutputHandler collects the output from an action and checks for failures in the ACTION_RUN_SUMMARY output
func ActionOutputHandler(outputs map[string]string, suppressOutput bool) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER ActionOutputHandler")
		var action *Action
		if ap, ok := plan.(ActionProvider); ok {
			action = ap.Action()
		} else {
			return fmt.Errorf("plan must implement ActionProvider for ActionOutputHandler")
		}
		lineHandlers := []lineHandler{
			actionOutputLineHandler(outputs, maps.Keys(action.Outputs.Variables)),
		}
		if suppressOutput {
			log.Ctx(ctx).Debug().Msgf("suppressing action output")
		} else {
			rawLogger := log.Ctx(ctx).With().Logger()
			lineHandlers = append(lineHandlers, func(s string) bool {
				s = strings.TrimRight(s, "\r\n")
				rawLogger.Info().Msg(s)
				return true
			})
		}
		logWriter := newLineWriter(lineHandlers...)
		log.Ctx(ctx).Debug().Msgf("Setting stdout/stderr to %+v", logWriter)
		plan.EnvironmentConfiguration().Stdout = logWriter
		plan.EnvironmentConfiguration().Stderr = logWriter

		if err := e(ctx); err != nil {
			maps.Clear(outputs)
			return err
		}
		for k, v := range outputs {
			if k == "ACTION_RUN_SUMMARY" {
				delete(outputs, k)
				// handle ACTION_RUN_SUMMARY output
				actionRunSummaries := make([]ActionRunSummaryMessage, 0)
				err := json.Unmarshal([]byte(v), &actionRunSummaries)
				if err != nil {
					return fmt.Errorf("unable to unmarshal ACTION_RUN_SUMMARY: %w\n%s", err, v)
				}
				var actionRunErrors error
				for _, actionRunSummary := range actionRunSummaries {
					if actionRunSummary.Level == ActionRunSummaryLevelError {
						actionRunErrors = errors.Join(
							actionRunErrors,
							fmt.Errorf("[%s] %s", actionRunSummary.Text, actionRunSummary.Message),
						)
					}
				}
				if actionRunErrors != nil {
					return actionRunErrors
				}
			}
		}
		log.Ctx(ctx).Debug().Msgf("action outputs: %+v", outputs)
		if len(outputs) > 0 {
			log.Ctx(ctx).Info().Msgf("")
			log.Ctx(ctx).Info().Msgf("💬 OUTPUTS:")
			for k, v := range outputs {
				log.Ctx(ctx).Info().Msgf("    %s = %s", k, v)
			}
			log.Ctx(ctx).Info().Msgf("")
		}
		log.Ctx(ctx).Debug().Msg("EXIT ActionOutputHandler")
		return nil
	}
}

var actionCommandPattern = regexp.MustCompile("^::([^ ]+)( (.+))?::([^\r\n]*)[\r\n]+$")

func actionOutputLineHandler(outputs map[string]string, filter []string) lineHandler {
	return func(line string) bool {
		if m := actionCommandPattern.FindStringSubmatch(line); m != nil {
			command := m[1]
			kvPairs := make(map[string]string)
			kvPairList := strings.Split(m[3], ",")
			for _, kvPair := range kvPairList {
				kv := strings.Split(kvPair, "=")
				if len(kv) == 2 {
					kvPairs[kv[0]] = kv[1]
				}
			}
			arg := m[4]
			if command == "set-output" {
				if slices.Contains(filter, kvPairs["name"]) || kvPairs["name"] == "ACTION_RUN_SUMMARY" {
					log.Debug().Msgf("Setting output %s = %s", kvPairs["name"], arg)
					outputs[kvPairs["name"]] = arg
				}
				return false
			}
		}
		return true
	}
}

// ActionRunSummaryMessage describes a messages that was returned from the action
type ActionRunSummaryMessage struct {
	Text              string                      // text of the message
	Level             ActionRunSummaryLevel       // level of the message
	Message           string                      // template to be used with TemplateVariables
	TemplateVariables []ActionRunTemplateVariable // variables to apply in the message template
}

// ActionRunSummaryLevel of an ActionRunSummaryMessage
type ActionRunSummaryLevel string

const (
	// ActionRunSummaryLevelError represents the error level
	ActionRunSummaryLevelError ActionRunSummaryLevel = "Error"
)

// ActionRunTemplateVariable describes a variable for a message template
type ActionRunTemplateVariable struct {
	Name  string
	Value string
}
