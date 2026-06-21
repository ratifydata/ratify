package sqlc

import (
	"log/slog"
	"os"
	"testing"

	"github.com/ratifydata/ratify/internal/testutil"
)

var testDB *testutil.TestContainer

func TestMain(m *testing.M) {
	//Stop the test in case an error is returned
	os.Exit(runTest(m))

}

func runTest(m *testing.M) int {
	var err error
	testDB, err = testutil.InitializePostgresContainer()
	if err != nil {
		slog.Error("failed to initialize test database")
		return 1
	}
	code := m.Run()
	testutil.TerminateContainer(testDB.Container)

	return code
}
