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
