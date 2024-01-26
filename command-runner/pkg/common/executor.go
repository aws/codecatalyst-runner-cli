package common

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"runtime/trace"
	"time"

	"github.com/rs/zerolog/log"
)

// Warning that implements `error` but safe to ignore.
type Warning struct {
	Message string
}

// Error the contract for error
func (w Warning) Error() string {
	return w.Message
}

// ErrDefer that implements `error` but safe to ignore.
var ErrDefer = errors.New("deferred")

// Executor define contract for the steps of a workflow
type Executor func(ctx context.Context) error

// Conditional define contract for the conditional predicate
type Conditional func(ctx context.Context) bool

// NewInfoExecutor is an executor that logs messages
func NewInfoExecutor(format string, args ...interface{}) Executor {
	return func(ctx context.Context) error {
		log.Ctx(ctx).Info().Msgf(format, args...)
		return nil
	}
}

// NewWarning returns a warning
func NewWarning(format string, args ...interface{}) error {
	return Warning{
		Message: fmt.Sprintf(format, args...),
	}
}

// NewWarningExecutor is an executor that returns a warning
func NewWarningExecutor(format string, args ...interface{}) Executor {
	return func(ctx context.Context) error {
		return NewWarning(format, args...)
	}
}

// NewDebugExecutor is an executor that logs messages
func NewDebugExecutor(format string, args ...interface{}) Executor {
	return func(ctx context.Context) error {
		log.Ctx(ctx).Debug().Msgf(format, args...)
		return nil
	}
}

// NewPipelineExecutor creates a new executor from a series of other executors
func NewPipelineExecutor(executors ...Executor) Executor {
	if len(executors) == 0 {
		return func(ctx context.Context) error {
			return nil
		}
	}
	var rtn Executor
	for _, executor := range executors {
		if rtn == nil {
			rtn = executor
		} else {
			rtn = rtn.Then(executor)
		}
	}
	return rtn
}

// NewConditionalExecutor creates a new executor based on conditions
func NewConditionalExecutor(conditional Conditional, trueExecutor Executor, falseExecutor Executor) Executor {
	return func(ctx context.Context) error {
		if conditional(ctx) {
			if trueExecutor != nil {
				return trueExecutor(ctx)
			}
		} else {
			if falseExecutor != nil {
				return falseExecutor(ctx)
			}
		}
		return nil
	}
}

// NewErrorExecutor creates a new executor that always errors out
func NewErrorExecutor(err error) Executor {
	return func(ctx context.Context) error {
		return err
	}
}

// NewParallelExecutor creates a new executor from a parallel of other executors
func NewParallelExecutor(parallel int, executors ...Executor) Executor {
	return func(ctx context.Context) error {
		work := make(chan Executor, len(executors))
		errs := make(chan error, len(executors))

		for i := 0; i < parallel; i++ {
			go func(lwork <-chan Executor, errs chan<- error) {
				for executor := range lwork {
					if err := executor(ctx); errors.Is(err, ErrDefer) {
						thisExecutor := executor
						go func() {
							time.Sleep(1 * time.Second)
							work <- thisExecutor
						}()
					} else {
						errs <- err
					}
				}
			}(work, errs)
		}

		for i := 0; i < len(executors); i++ {
			work <- executors[i]
		}
		defer close(work)

		// Executor waits all executors to cleanup these resources.
		var rtnError error
		for i := 0; i < len(executors); i++ {
			select {
			case err := <-errs:
				switch err.(type) {
				case Warning:
					log.Ctx(ctx).Debug().Err(err).Msg("Got warning")
				default:
					rtnError = errors.Join(rtnError, err)
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		log.Ctx(ctx).Debug().Err(rtnError).Msg("Parallel executor finished")

		return rtnError
	}
}

// Then runs another executor if this executor succeeds
func (e Executor) Then(then Executor) Executor {
	return func(ctx context.Context) error {
		err := e(ctx)
		if err != nil {
			switch err.(type) {
			case Warning:
				log.Ctx(ctx).Warn().Err(err)
			default:
				return err
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return then(ctx)
	}
}

// If only runs this executor if conditional is true
func (e Executor) If(conditional Conditional) Executor {
	return func(ctx context.Context) error {
		if conditional(ctx) {
			return e(ctx)
		}
		return nil
	}
}

// IfNot only runs this executor if conditional is true
func (e Executor) IfNot(conditional Conditional) Executor {
	return func(ctx context.Context) error {
		if !conditional(ctx) {
			return e(ctx)
		}
		return nil
	}
}

// IfBool only runs this executor if conditional is true
func (e Executor) IfBool(conditional bool) Executor {
	return e.If(func(ctx context.Context) bool {
		return conditional
	})
}

// Finally adds an executor to run after other executor
func (e Executor) Finally(finally Executor) Executor {
	return func(ctx context.Context) error {
		err := e(ctx)
		err2 := finally(ctx)
		if err2 != nil {
			return fmt.Errorf("Error occurred running finally: %v (original error: %v)", err2, err)
		}
		return err
	}
}

// Not return an inverted conditional
func (c Conditional) Not() Conditional {
	return func(ctx context.Context) bool {
		return !c(ctx)
	}
}

// Wrapper is a type that performs activities before and/or after an executor runs
type Wrapper func(ctx context.Context, e Executor) error

// Wrap returns a new [Executor] that applies this [Wrapper] to the provided [Executor]
func (w Wrapper) Wrap(e Executor) Executor {
	return func(ctx context.Context) error {
		return w(ctx, e)
	}
}

// WrapWith returns a new [Executor] that applies the provided [Wrapper]s to this [Executor]
func (e Executor) WrapWith(wrappers ...Wrapper) Executor {
	rtn := e
	for _, w := range wrappers {
		rtn = w.Wrap(rtn)
	}
	return rtn
}

// CatchPanic wraps the executor with panic handler
func CatchPanic(ctx context.Context, e Executor) error {
	err := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err <- fmt.Errorf("💀 panic: %v\n%s", r, debug.Stack())
			}
			close(err)
		}()
		err <- e(ctx)
	}()
	select {
	case rtn := <-err:
		return rtn
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CatchPanic wraps the executor with panic handler
func (e Executor) CatchPanic() Executor {
	return func(ctx context.Context) error {
		return CatchPanic(ctx, e)
	}
}

// TraceRegion wraps the executor with a trace
func (e Executor) TraceRegion(regionType string) Executor {
	return func(ctx context.Context) error {
		var err error
		trace.WithRegion(ctx, regionType, func() {
			err = e(ctx)
		})
		return err
	}
}

// ReadyFunc determines when an executor is ready to run
type ReadyFunc func() (bool, error)

// DeferUntil a ReadyFunc is ready
func (e Executor) DeferUntil(ready ReadyFunc) Executor {
	return func(ctx context.Context) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if ok, err := ready(); ok {
			return e(ctx)
		} else if err != nil {
			return err
		}
		return ErrDefer
	}
}
