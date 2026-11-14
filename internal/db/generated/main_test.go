// Copyright (C) 2026 Ratify Data
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
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
	//Initializes Ratify Internal Data And Schema
	testDB, err = testutil.InitializePostgresContainer()
	if err != nil {
		slog.Error("failed to initialize test database")
		return 1
	}
	code := m.Run()
	testutil.TerminateContainer(testDB.Internal.Container)
	testutil.TerminateContainer(testDB.External.Container)

	return code
}
