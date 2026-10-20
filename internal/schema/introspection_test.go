package schema

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"reflect"
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

			exists, err := TableExist(ctx, db, "public", tt.tableName)
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
	const tableName = "schema_snapshot_test"
	if _, err := db.ExecContext(t.Context(), `CREATE TABLE schema_snapshot_test (
 id integer PRIMARY KEY,
 required_name text NOT NULL,
 optional_name text
 )`); err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DROP TABLE schema_snapshot_test`); err != nil {
			t.Errorf("drop fixture: %v", err)
		}
	})

	t.Run("snapshot", func(t *testing.T) {
		got, err := GetTableSchema(t.Context(), db, "public", tableName)
		if err != nil {
			t.Fatalf("GetTableSchema(): %v", err)
		}
		want := &TableSnapshot{
			TableName:  tableName,
			SchemaName: "public",
			Columns: []TableData{
				{ColumnName: "id", DataType: "integer", IsPrimaryKey: true, IsNullable: false, OrdinalPosition: 1},
				{ColumnName: "required_name", DataType: "text", IsNullable: false, OrdinalPosition: 2},
				{ColumnName: "optional_name", DataType: "text", IsNullable: true, OrdinalPosition: 3},
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("snapshot = %+v, want %+v", got, want)
		}
	})
	t.Run("missing table", func(t *testing.T) {
		_, err := GetTableSchema(t.Context(), db, "public", "schema_snapshot_missing")
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("error = %v, want sql.ErrNoRows", err)
		}
	})
	t.Run("canceled query", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := GetTableSchema(ctx, db, "public", tableName)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("error = %v, want context.Canceled", err)
		}
	})
}

func TestSnapshotSchemaIsolationAndCompositeKey(t *testing.T) {
	db := testDB.External.DB
	if _, err := db.ExecContext(t.Context(), `
 CREATE SCHEMA snapshot_a;
 CREATE SCHEMA snapshot_b;
 CREATE TABLE snapshot_a.same_name (id integer);
 CREATE TABLE snapshot_b.same_name (tenant_id integer, id integer, note text, PRIMARY KEY (tenant_id, id));
 `); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DROP SCHEMA snapshot_a CASCADE; DROP SCHEMA snapshot_b CASCADE`); err != nil {
			t.Error(err)
		}
	})
	snapshot, err := GetTableSchema(t.Context(), db, "snapshot_b", "same_name")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SchemaName != "snapshot_b" || len(snapshot.Columns) != 3 {
		t.Fatalf("wrong snapshot: %+v", snapshot)
	}
	for i, column := range snapshot.Columns {
		if column.IsPrimaryKey != (i < 2) {
			t.Errorf("incorrect primary key: %+v", column)
		}
	}
	exists, err := TableExist(t.Context(), db, "public", "same_name")
	if err != nil || exists {
		t.Fatalf("existence leaked across schemas: %v, %v", exists, err)
	}
}
