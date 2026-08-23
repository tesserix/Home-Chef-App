package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateBackfillExecution_DryRunNeedsNoConfirmation(t *testing.T) {
	require.NoError(t, validateBackfillExecution(false, "", "production-project"))
}

func TestValidateBackfillExecution_ApplyRequiresExactProject(t *testing.T) {
	require.Error(t, validateBackfillExecution(true, "", "production-project"))
	require.Error(t, validateBackfillExecution(true, "different-project", "production-project"))
	require.NoError(t, validateBackfillExecution(true, "production-project", "production-project"))
}

func TestValidateBackfillExecution_RequiresKnownProject(t *testing.T) {
	require.Error(t, validateBackfillExecution(false, "", ""))
}
