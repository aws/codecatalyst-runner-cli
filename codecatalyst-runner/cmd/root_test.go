package cmd

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_RootCommandVersion(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	cmd := NewRootCmd("0.0.0")

	assert.Equal("0.0.0", cmd.Version)

	cmd.SetArgs([]string{"--version"})
	b := bytes.NewBufferString("")
	cmd.SetOut(b)

	err := cmd.ExecuteContext(ctx)
	assert.NoError(err)
	out, err := io.ReadAll(b)
	assert.NoError(err)
	assert.Equal("ccr version 0.0.0\n", string(out))
}

func Test_RootCommandFlags(t *testing.T) {
	assert := assert.New(t)

	cmd := NewRootCmd("0.0.0")

	var tests = []struct {
		name      string
		shortName string
		defValue  string
	}{
		{name: "verbose", shortName: "V", defValue: "false"},
	}

	for _, tt := range tests {
		flag := cmd.PersistentFlags().Lookup(tt.name)
		assert.NotNil(flag, "%s flag exits", tt.name)
		assert.Equal(tt.shortName, flag.Shorthand, "%s Shorthand", tt.name)
		assert.Equal(tt.defValue, flag.DefValue, "%s DefValue", tt.name)
	}
}
