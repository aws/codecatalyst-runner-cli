package common

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWorkflow(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	// empty
	emptyWorkflow := NewPipelineExecutor()
	assert.Nil(emptyWorkflow(ctx))

	// error case
	errorWorkflow := NewErrorExecutor(fmt.Errorf("test error"))
	assert.NotNil(errorWorkflow(ctx))

	// multiple success case
	runcount := 0
	successWorkflow := NewPipelineExecutor(
		func(ctx context.Context) error {
			runcount++
			return nil
		},
		func(ctx context.Context) error {
			runcount++
			return nil
		})
	assert.Nil(successWorkflow(ctx))
	assert.Equal(2, runcount)
}

func TestNewInfoExecutor(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	err := NewInfoExecutor("format: %s", "args")(ctx)
	assert.NoError(err)
}
func TestNewDebugExecutor(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	err := NewDebugExecutor("format: %s", "args")(ctx)
	assert.NoError(err)
}

func TestNewWarningExecutor(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	err := NewWarningExecutor("format: %s", "args")(ctx)
	assert.Error(err)
	assert.Equal(err.Error(), "format: args")
}

func TestThen(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	warn := NewWarningExecutor("warn")
	errorEx := NewErrorExecutor(fmt.Errorf("error"))
	info := NewInfoExecutor("info")

	warnErr := warn.Then(info)(ctx)
	assert.NoError(warnErr)

	errorErr := errorEx.Then(info)(ctx)
	assert.Error(errorErr)
	assert.Equal(errorErr.Error(), "error")
}

func TestFinally(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	errExec := NewErrorExecutor(fmt.Errorf("error"))

	var cnt = 0
	incrExec := Executor(func(ctx context.Context) error {
		cnt++
		return nil
	})

	incrErr := incrExec.Finally(incrExec)(ctx)
	assert.NoError(incrErr)
	assert.Equal(2, cnt)

	cnt = 0
	errorErr := errExec.Finally(incrExec)(ctx)
	assert.Error(errorErr)
	assert.Equal(errorErr.Error(), "error")
	assert.Equal(1, cnt)
}

func TestConditionals(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	errorEx := NewErrorExecutor(fmt.Errorf("error"))

	trueCond := func(ctx context.Context) bool {
		return true
	}
	falseCond := func(ctx context.Context) bool {
		return false
	}
	iftrue := errorEx.If(trueCond)(ctx)
	assert.Error(iftrue)
	iffalse := errorEx.If(falseCond)(ctx)
	assert.NoError(iffalse)
	iftrue = errorEx.IfBool(true)(ctx)
	assert.Error(iftrue)
	iffalse = errorEx.IfBool(false)(ctx)
	assert.NoError(iffalse)
	ifnottrue := errorEx.IfNot(trueCond)(ctx)
	assert.NoError(ifnottrue)
	ifnotfalse := errorEx.IfNot(falseCond)(ctx)
	assert.Error(ifnotfalse)
	ifnottrue = errorEx.If(Conditional(trueCond).Not())(ctx)
	assert.NoError(ifnottrue)
	ifnotfalse = errorEx.If(Conditional(falseCond).Not())(ctx)
	assert.Error(ifnotfalse)
}

func TestNewConditionalExecutor(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	trueCount := 0
	falseCount := 0

	err := NewConditionalExecutor(func(ctx context.Context) bool {
		return false
	}, func(ctx context.Context) error {
		trueCount++
		return nil
	}, func(ctx context.Context) error {
		falseCount++
		return nil
	})(ctx)

	assert.Nil(err)
	assert.Equal(0, trueCount)
	assert.Equal(1, falseCount)

	err = NewConditionalExecutor(func(ctx context.Context) bool {
		return true
	}, func(ctx context.Context) error {
		trueCount++
		return nil
	}, func(ctx context.Context) error {
		falseCount++
		return nil
	})(ctx)

	assert.Nil(err)
	assert.Equal(1, trueCount)
	assert.Equal(1, falseCount)
}

func TestNewParallelExecutor(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	count := 0
	activeCount := 0
	maxCount := 0
	emptyWorkflow := NewPipelineExecutor(func(ctx context.Context) error {
		count++

		activeCount++
		if activeCount > maxCount {
			maxCount = activeCount
		}
		time.Sleep(2 * time.Second)
		activeCount--

		return nil
	})

	err := NewParallelExecutor(2, emptyWorkflow, emptyWorkflow, emptyWorkflow)(ctx)

	assert.Equal(3, count, "should run all 3 executors")
	assert.Equal(2, maxCount, "should run at most 2 executors in parallel")
	assert.Nil(err)
}

func TestNewParallelExecutorFailed(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	count := 0
	errExpected := fmt.Errorf("fake error")
	successWorkflow := NewPipelineExecutor(func(ctx context.Context) error {
		count++
		return nil
	})
	errorWorkflow := NewPipelineExecutor(func(ctx context.Context) error {
		count++
		return errExpected
	})
	err := NewParallelExecutor(1, successWorkflow, errorWorkflow, successWorkflow)(ctx)
	assert.Equal(3, count)
	assert.Error(errExpected, err)
}

func TestNewParallelExecutorCanceled(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	count := 0
	successWorkflow := NewPipelineExecutor(func(ctx context.Context) error {
		count++
		return nil
	})
	err := NewParallelExecutor(2, successWorkflow, successWorkflow, successWorkflow, successWorkflow)(ctx)
	assert.Equal(0, count)
	assert.ErrorIs(context.Canceled, err)
}

func TestWrapper(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	var wrapE Executor
	sampleWrapper := func(ctx context.Context, e Executor) error {
		wrapE = e
		return nil
	}

	info := NewInfoExecutor("info")

	err := info.WrapWith(sampleWrapper)(ctx)
	assert.NoError(err)
	assert.NotNil(wrapE)
}
