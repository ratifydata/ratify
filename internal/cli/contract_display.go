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

package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/ratifydata/ratify/internal/contract"
	"github.com/spf13/cobra"
)

func displayContract(cmd *cobra.Command, item *contract.ContractModel) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	// Keep the first formatting/write error and stop writing after it.
	var writeErr error
	write := func(format string, args ...any) {
		if writeErr == nil {
			_, writeErr = fmt.Fprintf(w, format, args...)
		}
	}
	sla := "Not set"
	if item.FreshnessSlaHours.Valid {
		sla = fmt.Sprintf("%d hours", item.FreshnessSlaHours.Int32)
	}
	write("Contract:\t%s\nID:\t%s\nConnection:\t%s\nTable:\t%s.%s\nStatus:\t%s\nCurrent version:\t%d\nDescription:\t%s\nFreshness SLA:\t%s\n",
		item.DisplayName, item.ID.String(), item.ConnectionID.String(), item.SchemaName, item.TableName, item.Status, item.CurrentVersion, item.Description.String, sla)
	write("\nColumns (version %d)\n", item.CurrentVersion)
	var columns []contract.ContractColumn
	for _, version := range item.Versions {
		if version.Version == item.CurrentVersion {
			columns = version.Columns
			break
		}
	}
	if len(columns) == 0 {
		write("No columns recorded for the current version.\n")
	} else {
		write("POSITION\tNAME\tTYPE\tNULLABLE\tPRIMARY KEY\tDESCRIPTION\tCONSTRAINTS\n")
		for _, column := range columns {
			write("%d\t%s\t%s\t%t\t%t\t%s\t%s\n", column.Position, column.ColumnName, column.DataType, column.IsNullable, column.IsPrimaryKey, column.Description.String, column.Constraints)
		}
	}
	write("\nVersion history\n")
	if len(item.Versions) == 0 {
		write("No versions.\n")
	} else {
		write("VERSION\tID\tCREATED BY\tCHANGE SUMMARY\tCOLUMNS\n")
		for _, version := range item.Versions {
			write("%d\t%s\t%s\t%s\t%d\n", version.Version, version.ID.String(), version.User.String(), version.Description.String, len(version.Columns))
		}
	}
	if writeErr != nil {
		return writeErr
	}
	return w.Flush()
}
