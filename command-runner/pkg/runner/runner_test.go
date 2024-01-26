//go:build all || docker
// +build all docker

package runner

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRunAll(t *testing.T) {
	assert := assert.New(t)

	type TestParams struct {
		TestCase      string
		Plans         []Plan
		Features      []Feature
		ExpectedError string
	}

	for _, tt := range []*TestParams{
		{
			TestCase: "Empty",
		},
		{
			TestCase: "Basic",
			Plans: []Plan{
				&MockPlan{
					environmentConfiguration: EnvironmentConfiguration{
						WorkingDir: "testdata/workingdir/basic",
						Env: map[string]string{
							"FOO": "BAR",
						},
					},
					commandGroups: []*CommandGroup{
						{
							Commands: []Command{
								{"echo $FOO | grep BAR"},
							},
						},
					},
				},
			},
			Features: []Feature{
				testLogFeature(),
			},
		},
	} {

		// setup the code under test
		ctx := context.Background()

		// run the commands
		plansProvider := &MockPlansProvider{}
		plansProvider.On("Plans").Return(tt.Plans, nil)
		featuresProvider := &MockFeaturesProvider{}
		featuresProvider.On("Features", mock.Anything).Return(tt.Features, nil)
		err := RunAll(ctx, &RunAllParams{
			Namespace:     "mockns",
			Plans:         plansProvider,
			Features:      featuresProvider,
			ExecutionType: ExecutionTypeShell,
		})

		if tt.ExpectedError != "" {
			assert.EqualError(err, tt.ExpectedError, "%s - RunAll error", tt.TestCase)
		} else {
			assert.NoError(err, "%s - RunAll error", tt.TestCase)
		}
	}
}

type MockPlansProvider struct {
	mock.Mock
}

func (mpp *MockPlansProvider) Plans(_ context.Context) ([]Plan, error) {
	args := mpp.Called()
	return args.Get(0).([]Plan), args.Error(1)
}

type MockFeaturesProvider struct {
	mock.Mock
}

func (mfp *MockFeaturesProvider) Features(plan Plan) ([]Feature, error) {
	args := mfp.Called(plan)
	return args.Get(0).([]Feature), args.Error(1)
}

func testLogFeature() Feature {
	return func(ctx context.Context, plan Plan, e PlanExecutor) error {
		ctx = log.With().Logger().Output(&zerolog.ConsoleWriter{Out: os.Stdout}).WithContext(ctx)
		return e(ctx)
	}
}
