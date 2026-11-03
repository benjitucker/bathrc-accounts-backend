package db

import "testing"

func TestTableName_DefaultIsUnchanged(t *testing.T) {
	SetTableNamePrefix("")
	if got := tableName("TrainingSubmissions"); got != "TrainingSubmissions" {
		t.Fatalf("expected production table name, got %q", got)
	}
}

func TestTableName_WithPrefix(t *testing.T) {
	SetTableNamePrefix("dev-")
	t.Cleanup(func() { SetTableNamePrefix("") })

	if got := tableName("Members"); got != "dev-Members" {
		t.Fatalf("expected dev-Members, got %q", got)
	}
}
