package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDryRun(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	assert.False(Dryrun(ctx))

	ctx = WithDryrun(ctx, true)
	assert.True(Dryrun(ctx))
}
