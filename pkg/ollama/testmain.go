package ollama

import (
	"os"
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	// verify no goroutine leaks across tests in this package
	defer goleak.VerifyTestMain(m)
	os.Exit(m.Run())
}
