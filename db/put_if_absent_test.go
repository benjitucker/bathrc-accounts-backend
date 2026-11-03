package db

import "testing"

func TestPutItemIfAbsent(t *testing.T) {
	createTestTable(t, testTable)
	tbl := newDBTable(t, testTable)

	first := &TestItem{Name: "Alice", Age: 30}
	first.SetID("user-1")

	created, err := putItemIfAbsent(tbl, first)
	if err != nil {
		t.Fatalf("first putItemIfAbsent failed: %v", err)
	}
	if !created {
		t.Fatalf("expected the first write to create the item")
	}

	second := &TestItem{Name: "Bob", Age: 40}
	second.SetID("user-1")

	created, err = putItemIfAbsent(tbl, second)
	if err != nil {
		t.Fatalf("second putItemIfAbsent returned an error: %v", err)
	}
	if created {
		t.Fatalf("expected the second write to be refused")
	}

	got, err := getItem[*TestItem](tbl, "user-1")
	if err != nil {
		t.Fatalf("getItem failed: %v", err)
	}
	if got.Name != "Alice" || got.Age != 30 {
		t.Fatalf("expected the original item to be kept, got %#v", got)
	}
}
