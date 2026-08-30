package schema

import (
	"context"
	"testing"
)

func TestTableExist(t *testing.T) {
	ctx := context.Background()
	//Creates DB in the test pool
	db := testDB.External.DB
	tests := []struct {
		testName  string
		tableName string
		simError  bool
		want      bool
		wantErr   bool
	}{
		{
			testName:  "Test_Table_Exist",
			tableName: "test_table_1",
			simError:  false,
			want:      true,
			wantErr:   false,
		}, {
			testName:  "Test_Table_Does_Not_Exist",
			tableName: "unavailable_table",
			simError:  false,
			want:      false,
			wantErr:   false,
		}, {
			testName:  "Test_Table_Query_Error",
			tableName: "test_table_1",
			simError:  true,
			want:      false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if tt.simError { //Simulates Error by Closing the DB
				db.Close()
			}
			exists, err := TableExist(ctx, db, tt.tableName)
			if (err != nil) != tt.wantErr {
				t.Errorf("TableExist() error = %v, wantErr %v", err, tt.wantErr)
			}
			if exists != tt.want {
				t.Errorf("TableExist() got = %v, want %v", exists, tt.want)
			}

		})
	}
}

func TestListTable(t *testing.T) {
	ctx := context.Background()
	db := testDB.External.DB
	tests := []struct {
		testName string
		wantSize int
		simError bool
		wantErr  bool
	}{
		{
			testName: "List_All_Tables",
			wantSize: 2,
			simError: false,
			wantErr:  false,
		}, {
			testName: "Empty_List_Of_Tables",
			wantSize: 0,
			simError: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if tt.simError {
				db.Exec(`DROP TABLE IF EXISTS test_table_1;
				DROP TABLE IF EXISTS test_table_2;`)
			}
			tableList, err := ListTables(ctx, db)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListTables() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(tableList) != tt.wantSize {
				t.Errorf("ListTables() size got = %v, want %v", len(tableList), tt.wantSize)
			}
		})
	}
}

func TestGetTableSchema(t *testing.T) {
	ctx := context.Background()
	db := testDB.External.DB
	tests := []struct {
		testName string
		wantSize int
		simError bool
		wantErr  bool
	}{
		{
			testName: "Get_Table_Schema_Success",
			wantSize: 2,
			simError: false,
			wantErr:  false,
		}, {
			testName: "Get_Table_Schema_With_No_Columns",
			wantSize: 0,
			simError: true,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if tt.simError {
				db.Exec(`ALTER TABLE  IF EXISTS test_table_1 DROP COLUMN id;
				ALTER TABLE  IF EXISTS test_table_1 DROP COLUMN test_column;`)
			}
			tableData, err := GetTableSchema(ctx, db, "test_table_1")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTableSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(tableData) != tt.wantSize {
				t.Errorf("GetTableSchema() got = %v, want %v", len(tableData), tt.wantSize)
			}
		})
	}

}
