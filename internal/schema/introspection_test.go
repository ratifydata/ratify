package schema

import (
	"context"
	"database/sql"
	"net/url"
	"testing"
)

func TestTableExist(t *testing.T) {
	db := testDB.External.DB
	if _, err := db.ExecContext(t.Context(), `
		CREATE TABLE IF NOT EXISTS test_table_1 (
			id serial PRIMARY KEY,
			test_column varchar(50) NOT NULL
		)`); err != nil {
		t.Fatalf("create table fixture: %v", err)
	}

	tests := []struct {
		name      string
		tableName string
		cancel    bool
		want      bool
		wantErr   bool
	}{
		{
			name:      "Test_Table_Exist",
			tableName: "test_table_1",
			want:      true,
		},
		{
			name:      "Test_Table_Does_Not_Exist",
			tableName: "unavailable_table",
		},
		{
			name:      "Test_Table_Query_Error",
			tableName: "test_table_1",
			cancel:    true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			if tt.cancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			exists, err := TableExist(ctx, db, tt.tableName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("TableExist() error = %v, wantErr %v", err, tt.wantErr)
			}
			if exists != tt.want {
				t.Errorf("TableExist() got = %v, want %v", exists, tt.want)
			}
		})
	}
}

func TestListTables(t *testing.T) {
	db := testDB.External.DB
	if _, err := db.ExecContext(t.Context(), `
		CREATE TABLE IF NOT EXISTS test_table_1 (
			id serial PRIMARY KEY,
			test_column varchar(50) NOT NULL
		);
		CREATE TABLE IF NOT EXISTS test_table_2 (
			id serial PRIMARY KEY,
			test_column varchar(50) NOT NULL
		)`); err != nil {
		t.Fatalf("create table fixtures: %v", err)
	}

	t.Run("List_All_Tables", func(t *testing.T) {
		tables, err := ListTables(t.Context(), db)
		if err != nil {
			t.Fatalf("ListTables() error = %v, want nil", err)
		}

		want := map[TableList]bool{
			{TableSchema: "public", TableName: "test_table_1"}: false,
			{TableSchema: "public", TableName: "test_table_2"}: false,
		}
		for _, table := range tables {
			if _, ok := want[table]; ok {
				want[table] = true
			}
		}
		for table, found := range want {
			if !found {
				t.Errorf("ListTables() did not return %+v; got %+v", table, tables)
			}
		}
	})

	t.Run("Empty_List_Of_Tables", func(t *testing.T) {
		const databaseName = "schema_introspection_empty"
		if _, err := db.ExecContext(t.Context(), `CREATE DATABASE schema_introspection_empty`); err != nil {
			t.Fatalf("create empty database: %v", err)
		}

		emptyDSN, err := url.Parse(testDB.External.DSN)
		if err != nil {
			t.Fatalf("parse external database DSN: %v", err)
		}
		emptyDSN.Path = "/" + databaseName
		emptyDB, err := sql.Open("pgx", emptyDSN.String())
		if err != nil {
			t.Fatalf("open empty database: %v", err)
		}
		t.Cleanup(func() {
			if err := emptyDB.Close(); err != nil {
				t.Errorf("close empty database: %v", err)
			}
			if _, err := db.ExecContext(context.Background(), `DROP DATABASE IF EXISTS schema_introspection_empty`); err != nil {
				t.Errorf("drop empty database: %v", err)
			}
		})

		tables, err := ListTables(t.Context(), emptyDB)
		if err != nil {
			t.Fatalf("ListTables() error = %v, want nil", err)
		}
		if len(tables) != 0 {
			t.Errorf("ListTables() returned %d tables, want 0: %+v", len(tables), tables)
		}
	})
}

func TestGetTableSchema(t *testing.T) {
	db := testDB.External.DB
	const tableName = "schema_columns_test"
	if _, err := db.ExecContext(t.Context(), `
		DROP TABLE IF EXISTS schema_columns_test;
		CREATE TABLE schema_columns_test (
			id serial PRIMARY KEY,
			test_column varchar(50) NOT NULL
		)`); err != nil {
		t.Fatalf("create schema fixture: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(),
			`DROP TABLE IF EXISTS schema_columns_test`); err != nil {
			t.Errorf("drop schema fixture: %v", err)
		}
	})

	tests := []struct {
		name      string
		tableName string
		wantSize  int
	}{
		{
			name:      "Get_Table_Schema_Success",
			tableName: tableName,
			wantSize:  2,
		},
		{
			name:      "Get_Table_Schema_For_Missing_Table",
			tableName: "schema_columns_test_missing",
			wantSize:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			columns, err := GetTableSchema(t.Context(), db, tt.tableName)
			if err != nil {
				t.Fatalf("GetTableSchema() error = %v, want nil", err)
			}
			if len(columns) != tt.wantSize {
				t.Errorf("GetTableSchema() got %d columns, want %d", len(columns), tt.wantSize)
			}
		})
	}
}
