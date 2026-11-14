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
package util

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
)

func ValidateOrgId(ctx context.Context) (pgtype.UUID, error) {
	orgID, ok := ctx.Value("OrgID").(pgtype.UUID)
	if !ok {
		slog.Error("OrgID missing from context")
		return pgtype.UUID{}, fmt.Errorf("OrgID missing from context")
	}
	if !orgID.Valid {
		slog.Error("Invalid orgId")
		return pgtype.UUID{}, errors.New("OrgID not valid")
	}
	return orgID, nil

}

func ValidateUserId(ctx context.Context) (pgtype.UUID, error) {
	userID, ok := ctx.Value("UserID").(pgtype.UUID)
	if !ok {
		slog.Error("UserID missing from context")
		return pgtype.UUID{}, fmt.Errorf("UserID missing from context")
	}
	if !userID.Valid {
		slog.Error("UserID not valid")
		return pgtype.UUID{}, errors.New("UserID not valid")
	}
	return userID, nil
}
