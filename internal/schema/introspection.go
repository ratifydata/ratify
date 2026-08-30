package schema

import (
	"context"
	"database/sql"
	"log/slog"
)

type TableList struct {
	TableSchema string
	TableName   string
}

type TableData struct {
	ColumnName string
	DataType   string
	IsNullable string
}

func ListTables(ctx context.Context, db *sql.DB) ([]TableList, error) {
	const user_table_queries = `SELECT table_schema, table_name 
	FROM information_schema.tables 
	WHERE table_type = 'BASE TABLE' 
	  AND table_schema NOT IN ('pg_catalog', 'information_schema');
	`
	var tList []TableList
	row, err := db.QueryContext(ctx, user_table_queries)
	defer func(row *sql.Rows) {
		err = row.Close()
		if err != nil {
			slog.Error("failed to close user table", "err", err)
		}
	}(row)

	if err != nil {
		return nil, err
	}

	for row.Next() {
		var t TableList
		if err := row.Scan(&t.TableSchema, &t.TableName); err != nil {
			return nil, err
		}
		tList = append(tList, t)
	}
	if err := row.Err(); err != nil {
		return nil, err
	}
	return tList, nil
}

func TableExist(ctx context.Context, db *sql.DB, table string) (bool, error) {
	const table_exist = `SELECT EXISTS (
    SELECT 1 
    FROM information_schema.tables 
    WHERE  table_name = $1);`

	var exists bool
	err := db.QueryRowContext(ctx, table_exist, table).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func GetTableSchema(ctx context.Context, db *sql.DB, tableName string) ([]TableData, error) {
	const table_column_query = `SELECT 
    	column_name,  data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = $1 ;`

	var columnList []TableData
	row, err := db.QueryContext(ctx, table_column_query, tableName)
	defer row.Close()
	if err != nil {
		return nil, err
	}
	for row.Next() {
		var t TableData
		if err := row.Scan(&t.ColumnName, &t.DataType, &t.IsNullable); err != nil {
			return nil, err
		}
		columnList = append(columnList, t)
	}
	if err := row.Err(); err != nil {

		return nil, err
	}
	return columnList, nil

}
