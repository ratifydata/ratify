package contract

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ratifydata/ratify/internal/schema"
)

func TestDiffSchema(t *testing.T) {
	id := schema.TableData{ColumnName: "id", DataType: "integer", OrdinalPosition: 1}
	note := schema.TableData{ColumnName: "note", DataType: "text", IsNullable: true, OrdinalPosition: 2}
	metadataChanged := id
	metadataChanged.IsPrimaryKey = true
	metadataChanged.OrdinalPosition = 3
	renamed := note
	renamed.ColumnName = "description"
	changed := id
	changed.DataType = "bigint"
	changed.IsNullable = true
	for _, tt := range []struct {
		name          string
		before, after []schema.TableData
		want          []SnapshotConflict
	}{
		{"matching", []schema.TableData{id, note}, []schema.TableData{id, note}, []SnapshotConflict{}},
		{"empty", nil, nil, []SnapshotConflict{}},
		{"added columns ignored", []schema.TableData{id}, []schema.TableData{id, note}, []SnapshotConflict{}},
		{"removed", []schema.TableData{id, note}, []schema.TableData{id}, []SnapshotConflict{{ColumnName: "note", Mismatch: ColumnRemoved, Before: &note}}},
		{"multiple changes", []schema.TableData{id, note}, []schema.TableData{changed}, []SnapshotConflict{
			{ColumnName: "id", Mismatch: ColumnTypeChanged, Before: &id, After: &changed},
			{ColumnName: "id", Mismatch: ColumnNullabilityChanged, Before: &id, After: &changed},
			{ColumnName: "note", Mismatch: ColumnRemoved, Before: &note},
		}},
		{"metadata ignored", []schema.TableData{id}, []schema.TableData{metadataChanged}, []SnapshotConflict{}},
		{"renamed column removed", []schema.TableData{note}, []schema.TableData{renamed}, []SnapshotConflict{{ColumnName: "note", Mismatch: ColumnRemoved, Before: &note}}},
		{"order ignored", []schema.TableData{id, note}, []schema.TableData{note, id}, []SnapshotConflict{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := &schema.TableSnapshot{Columns: tt.before}
			after := &schema.TableSnapshot{Columns: tt.after}
			got, err := DiffSchema(before, after)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("diff = %+v, want %+v", got, tt.want)
			}
			data, err := json.Marshal(before)
			if err != nil {
				t.Fatal(err)
			}
			validated, err := ValidateContractAgainstSchema(data, after)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(validated, tt.want) {
				t.Fatalf("validation = %+v, want %+v", validated, tt.want)
			}
		})
	}
}

func TestDiffSchemaIndividualChanges(t *testing.T) {
	before := schema.TableData{ColumnName: "name", DataType: "text", IsNullable: true}
	for _, tt := range []struct {
		name  string
		after schema.TableData
		kind  SchemaChange
	}{
		{"type", schema.TableData{ColumnName: "name", DataType: "character varying", IsNullable: true}, ColumnTypeChanged},
		{"nullable to required", schema.TableData{ColumnName: "name", DataType: "text"}, ColumnNullabilityChanged},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DiffSchema(&schema.TableSnapshot{Columns: []schema.TableData{before}}, &schema.TableSnapshot{Columns: []schema.TableData{tt.after}})
			if err != nil {
				t.Fatal(err)
			}
			want := []SnapshotConflict{{ColumnName: "name", Mismatch: tt.kind, Before: &before, After: &tt.after}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestValidationNilLiveSnapshot(t *testing.T) {
	before := &schema.TableSnapshot{}
	data, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name     string
		validate func() ([]SnapshotConflict, error)
	}{
		{"diff", func() ([]SnapshotConflict, error) { return DiffSchema(before, nil) }},
		{"validate", func() ([]SnapshotConflict, error) { return ValidateContractAgainstSchema(data, nil) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.validate()
			if err == nil || got != nil {
				t.Fatalf("got %v, %v; want error", got, err)
			}
		})
	}
}

func TestValidateContractInvalidJSON(t *testing.T) {
	for _, input := range []string{"", "{", "[]", `{"columns":"invalid"}`, `{"columns":[{"IsNullable":"YES"}]}`} {
		t.Run(input, func(t *testing.T) {
			got, err := ValidateContractAgainstSchema([]byte(input), &schema.TableSnapshot{})
			if err == nil || got != nil {
				t.Fatalf("got %v, %v; want error", got, err)
			}
		})
	}
}
