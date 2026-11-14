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
package contract

import (
	"encoding/json"
	"fmt"

	"github.com/ratifydata/ratify/internal/schema"
)

// SchemaChange describes a change from the persisted contract to the live schema.
type SchemaChange string

const (
	ColumnRemoved            SchemaChange = "removed"
	ColumnTypeChanged        SchemaChange = "type_changed"
	ColumnNullabilityChanged SchemaChange = "nullability_changed"
)

// SnapshotConflict includes both definitions so callers can display the mismatch.
// Before is nil for additions; After is nil for removals.
type SnapshotConflict struct {
	ColumnName string            `json:"column_name"`
	Mismatch   SchemaChange      `json:"mismatch"`
	Before     *schema.TableData `json:"before"`
	After      *schema.TableData `json:"after"`
}

// ValidateContractAgainstSchema compares persisted snapshot JSON with a live
// snapshot. Differences are returned as data, while invalid inputs return errors.
func ValidateContractAgainstSchema(contractSchema []byte, live *schema.TableSnapshot) ([]SnapshotConflict, error) {
	var persisted *schema.TableSnapshot
	if err := json.Unmarshal(contractSchema, &persisted); err != nil {
		return nil, fmt.Errorf("decode contract snapshot: %w", err)
	}
	return DiffSchema(persisted, live)
}

// DiffSchema compares columns by name, ignoring order and primary-key metadata.
// Changes follow persisted column order, then additions follow live column order.
// Renaming a column is represented as a removal and an addition.
func DiffSchema(prior, current *schema.TableSnapshot) ([]SnapshotConflict, error) {
	currentSnapshotColumns, err := indexSnapshot(current)
	if err != nil {
		return nil, err
	}
	conflicts := make([]SnapshotConflict, 0)
	for _, old := range prior.Columns {
		current, exists := currentSnapshotColumns[old.ColumnName]
		if !exists {
			conflicts = append(conflicts, SnapshotConflict{ColumnName: old.ColumnName, Mismatch: ColumnRemoved, Before: &old})
			continue
		}
		if old.DataType != current.DataType {
			conflicts = append(conflicts, SnapshotConflict{ColumnName: old.ColumnName, Mismatch: ColumnTypeChanged, Before: &old, After: &current})
		}
		if old.IsNullable != current.IsNullable {
			conflicts = append(conflicts, SnapshotConflict{ColumnName: old.ColumnName, Mismatch: ColumnNullabilityChanged, Before: &old, After: &current})
		}
	}
	return conflicts, nil
}

func indexSnapshot(snapshot *schema.TableSnapshot) (map[string]schema.TableData, error) {
	if snapshot == nil {
		return nil, fmt.Errorf(" snapshot is nil")
	}
	columns := make(map[string]schema.TableData, len(snapshot.Columns))
	for _, column := range snapshot.Columns {
		columns[column.ColumnName] = column
	}
	return columns, nil
}
