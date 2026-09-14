// Package testenv gates the tests that are not hermetic.
package testenv

import (
	"os"
	"testing"
)

// IntegrationEnv opts a run in to the tests that touch the developer's actual
// session.
const IntegrationEnv = "HYPRVOICE_TEST_INTEGRATION"

// RequireIntegration skips the calling test unless integration tests have been
// explicitly enabled.
//
// A few tests in this repo are not hermetic: they type into whatever window
// currently has focus, raise desktop notifications, and open the microphone.
// They were previously skipped only when CI=true, which meant the default for
// anyone running `go test ./...` on their own machine was to have those side
// effects -- stray text typed into whatever they happened to be looking at.
//
// Opting in rather than out puts the surprising behaviour behind a deliberate
// choice, and keeps CI green either way since the variable is simply unset
// there.
func RequireIntegration(t *testing.T) {
	t.Helper()

	if os.Getenv(IntegrationEnv) == "" {
		t.Skipf("skipping integration test: it types into the focused window, "+
			"sends desktop notifications, or records audio. Set %s=1 to run it.",
			IntegrationEnv)
	}
}
