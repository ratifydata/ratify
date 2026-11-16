package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/ratifydata/ratify/internal/contract"
	"github.com/spf13/cobra"
)

func displayContract(cmd *cobra.Command, item *contract.ContractModel) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	// Buffer the complete view so write failures are returned by Flush.
	sla := "Not set"
	if item.FreshnessSlaHours.Valid {
		sla = fmt.Sprintf("%d hours", item.FreshnessSlaHours.Int32)
	}
	fmt.Fprintf(w, "Contract:\t%s\nID:\t%s\nConnection:\t%s\nTable:\t%s.%s\nStatus:\t%s\nCurrent version:\t%d\nDescription:\t%s\nFreshness SLA:\t%s\n",
		item.DisplayName, item.ID.String(), item.ConnectionID.String(), item.SchemaName, item.TableName, item.Status, item.CurrentVersion, item.Description.String, sla)
	fmt.Fprintf(w, "\nColumns (version %d)\n", item.CurrentVersion)
	var columns []contract.ContractColumn
	for _, version := range item.Versions {
		if version.Version == item.CurrentVersion {
			columns = version.Columns
			break
		}
	}
	if len(columns) == 0 {
		fmt.Fprintln(w, "No columns recorded for the current version.")
	} else {
		fmt.Fprintln(w, "POSITION\tNAME\tTYPE\tNULLABLE\tPRIMARY KEY\tDESCRIPTION\tCONSTRAINTS")
		for _, column := range columns {
			fmt.Fprintf(w, "%d\t%s\t%s\t%t\t%t\t%s\t%s\n", column.Position, column.ColumnName, column.DataType, column.IsNullable, column.IsPrimaryKey, column.Description.String, column.Constraints)
		}
	}
	fmt.Fprintln(w, "\nVersion history")
	if len(item.Versions) == 0 {
		fmt.Fprintln(w, "No versions.")
	} else {
		fmt.Fprintln(w, "VERSION\tID\tCREATED BY\tCHANGE SUMMARY\tCOLUMNS")
		for _, version := range item.Versions {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\n", version.Version, version.ID.String(), version.User.String(), version.Description.String, len(version.Columns))
		}
	}
	return w.Flush()
}
