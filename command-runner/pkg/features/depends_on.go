package features

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// ProgressHandle handles callbacks from the progress on a plan execution
type ProgressHandle interface {
	Success()                                  // success is called when the handle completes successfully
	Failure(err error)                         // failure is called when the handle completes unsuccessfully
	IsReady(dependsOn ...string) (bool, error) // isReady is called to determine if the handle is ready
}

// DependsOn waits for dependencies
func DependsOn(progressHandle ProgressHandle) runner.Feature {
	logged := make([]string, 0)
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER DependsOn")
		for _, dependsOn := range plan.DependsOn() {
			if ready, err := progressHandle.IsReady(dependsOn); err != nil {
				log.Ctx(ctx).Err(err).Msgf("❌ aborted while waiting for %s", dependsOn)
				progressHandle.Failure(err)
				return err
			} else if !ready {
				if !slices.Contains(logged, dependsOn) {
					log.Ctx(ctx).Info().Msgf("⏳ WAITING for %s to succeed", dependsOn)
					logged = append(logged, dependsOn)
				}
				return common.ErrDefer
			}
		}
		err := e(ctx)
		if err != nil {
			if !errors.Is(err, common.ErrDefer) {
				progressHandle.Failure(err)
			}
			return err
		}
		progressHandle.Success()
		log.Ctx(ctx).Debug().Msg("EXIT DependsOn")
		return err
	}
}

// PlanTracker provides [ProgressHandle] for each plan and tracks progress across all plans
type PlanTracker struct {
	pending []string
	failed  []string
	mu      sync.Mutex
}

type progressHandle struct {
	pt     *PlanTracker
	planID string
}

// ProgressHandle returns a [ProgressHandle] for the given plan.
func (pt *PlanTracker) ProgressHandle(planID string) ProgressHandle {
	pt.pending = append(pt.pending, planID)
	return &progressHandle{
		pt:     pt,
		planID: planID,
	}
}

func (ph *progressHandle) Success() {
	ph.pt.mu.Lock()
	defer ph.pt.mu.Unlock()
	newPending := make([]string, 0)
	for _, p := range ph.pt.pending {
		if p != ph.planID {
			newPending = append(newPending, p)
		}
	}
	ph.pt.pending = newPending
}
func (ph *progressHandle) Failure(_ error) {
	ph.pt.mu.Lock()
	defer ph.pt.mu.Unlock()
	ph.pt.failed = append(ph.pt.failed, ph.planID)
	newPending := make([]string, 0)
	for _, p := range ph.pt.pending {
		if p != ph.planID {
			newPending = append(newPending, p)
		}
	}
	ph.pt.pending = newPending
}
func (ph *progressHandle) IsReady(dependsOn ...string) (bool, error) {
	ready := true
	var group string
	if strings.Contains(ph.planID, "@") {
		parts := strings.Split(ph.planID, "@")
		group = parts[0]
	}
	for _, dependency := range dependsOn {
		for _, f := range ph.pt.failed {
			if f == dependency || f == fmt.Sprintf("%s@%s", group, dependency) || strings.HasPrefix(f, fmt.Sprintf("%s@", dependency)) {
				return false, common.NewWarning("cancelled %s: dependency %s failed", ph.planID, dependency)
			}
		}
		for _, p := range ph.pt.pending {
			if p == dependency || p == fmt.Sprintf("%s@%s", group, dependency) || strings.HasPrefix(p, fmt.Sprintf("%s@", dependency)) {
				ready = false
				log.Debug().Msgf("DEFER [%s] for dependency [%s]", ph.planID, p)
				break
			}
		}
	}
	return ready, nil
}
