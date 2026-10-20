package schema

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
)

type TableList struct {
	TableSchema string
	TableName   string
}

type TableData struct {
	ColumnName      string `json:"ColumnName"`
	DataType        string `json:"DataType"`
	IsNullable      bool   `json:"IsNullable"`
	OrdinalPosition int32  `json:"OrdinalPosition"`
}

type TableSnapshot struct {
	TableName string      `json:"table_name"`
	Columns   []TableData `json:"columns"`
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

func GetTableSchema(ctx context.Context, db *sql.DB, tableName string) (*TableSnapshot, error) {
	const table_column_query = `SELECT jsonb_build_object(
    'TableName', t.table_name,
    'SchemaName', t.table_schema,
    'Columns', (
        SELECT COALESCE(
            jsonb_agg(
                jsonb_build_object(
                    'ColumnName', c.column_name,
                    'DataType', c.data_type,
                    'IsNullable', c.is_nullable,
                	'OrdinalPosition', c.ordinal_position
                )
                ORDER BY c.ordinal_position
            ),
            '[]'::jsonb
        )
        FROM information_schema.columns c
        WHERE c.table_schema = t.table_schema
          AND c.table_name = t.table_name
    )
) AS table_details
FROM information_schema.tables t
WHERE t.table_type = 'BASE TABLE'
  AND t.table_schema NOT IN (
      'pg_catalog',
      'information_schema'
  )
  AND t.table_schema NOT LIKE 'pg_toast%'
ORDER BY t.table_schema, t.table_name;`

	var snapShot TableSnapshot
	row, err := db.QueryContext(ctx, table_column_query, tableName)
	defer func(row *sql.Rows) {
		err := row.Close()
		if err != nil {
			slog.Error("failed to close table schema", "err", err)
		}
	}(row)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(table_column_query), &snapShot); err != nil {
		return nil, err
	}
	return &snapShot, nil

}
