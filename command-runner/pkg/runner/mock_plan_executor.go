package runner

import (
	"context"
	"os"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/stretchr/testify/mock"
)

// MockPlanExecutor provides a mechanism to test [Feature] execution
type MockPlanExecutor struct {
	mock.Mock
	plan Plan
	ex   common.Executor
}

// WithPlan uses the provided plan during execution
func (m *MockPlanExecutor) WithPlan(plan Plan) *MockPlanExecutor {
	m.plan = plan
	return m
}

// WithExecutor allows you to test a feature with a given executor providing behaviors wrapped by the Feature
func (m *MockPlanExecutor) WithExecutor(executor common.Executor) *MockPlanExecutor {
	m.ex = executor
	return m
}

// OnExecute provides a mechanism to mock the call to Execute()
func (m *MockPlanExecutor) OnExecute(ctx interface{}) *mock.Call {
	return m.On("Execute", ctx)
}

// Execute executes the provided feature with the provided plan
func (m *MockPlanExecutor) Execute(ctx context.Context, feature Feature) error {
	return feature(ctx, m.plan, m.executor())
}

func (m *MockPlanExecutor) executor() PlanExecutor {
	return func(ctx context.Context) error {
		args := m.MethodCalled("Execute", ctx)
		if m.ex != nil {
			return m.ex(ctx)
		}
		return args.Error(0)
	}
}

// MockPlan provides a mock plan
type MockPlan struct {
	id                       string
	environmentConfiguration EnvironmentConfiguration
	commandGroups            []*CommandGroup
	dependsOn                []string
}

// EnvironmentConfiguration provides access to the [EnviromentConfiguration] for this plan
func (mp *MockPlan) EnvironmentConfiguration() *EnvironmentConfiguration {
	if mp.environmentConfiguration.Stdout == nil {
		mp.environmentConfiguration.Stdout = os.Stdout
	}
	if mp.environmentConfiguration.Stderr == nil {
		mp.environmentConfiguration.Stderr = os.Stderr
	}
	return &mp.environmentConfiguration
}

// CommandGroups provides access to the [CommandGroups] for this plan
func (mp *MockPlan) CommandGroups() []*CommandGroup {
	if mp.commandGroups == nil {
		mp.commandGroups = make([]*CommandGroup, 0)
	}
	return mp.commandGroups
}

// ID provides access to the ID for this plan
func (mp *MockPlan) ID() string {
	return mp.id
}

// DependsOn provides access to the [DependsOn] for this plan
func (mp *MockPlan) DependsOn() []string {
	return mp.dependsOn
}

// AddDependsOn provides access to add a new dependency
func (mp *MockPlan) AddDependsOn(dependencies ...string) {
	mp.dependsOn = append(mp.dependsOn, dependencies...)
}

// WithID allows you to configure the ID of this plan
func (mp *MockPlan) WithID(id string) *MockPlan {
	mp.id = id
	return mp
}

// AddCommandGroup provides access to add a new commandGroup
func (mp *MockPlan) AddCommandGroup(commandGroups ...*CommandGroup) *MockPlan {
	mp.commandGroups = append(mp.commandGroups, commandGroups...)
	return mp
}
