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
package apperrors

import (
	"errors"
	"testing"
)

func TestErrorsPreserveCause(t *testing.T) {
	cause := errors.New("underlying failure")
	for _, err := range []error{
		&NotFoundError{Err: cause},
		&BadRequestError{Err: cause},
		&ConflictError{Err: cause},
	} {
		if err.Error() != cause.Error() {
			t.Errorf("message = %q", err.Error())
		}
		if !errors.Is(err, cause) {
			t.Errorf("%T does not preserve cause", err)
		}
	}
}
