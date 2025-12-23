package main

import (
	"strings"
	"testing"
)

var sampleCSV = []byte(`"First Name","Last Name","Date of Birth","Sex at Birth","Email Address","Individual Membership Member No.","BATH RIDING CLUB Membership Status","BATH RIDING CLUB Membership Valid From","BATH RIDING CLUB Membership Valid To","BATH RIDING CLUB Membership Membership Type"
Alice,Test,1987-01-23,Female,alice.test@example.com,99703368,Current,2025-06-13,2026-06-13,Senior
Bob,Example,,,bob.example@example.com,23023131,,,,
Charlie,Sample,2012-01-23,Male,charlie.sample@example.com,99692206,Lapsed,2024-01-03,2025-01-03,Junior`)

func TestParseMembersCSV(t *testing.T) {
	members, err := parseMembersCSV(sampleCSV)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(members))
	}

	// Test first member (full data)
	first := members[0]
	if first.FirstName != "Alice" || first.LastName != "Test" {
		t.Errorf("unexpected first member name: %s %s", first.FirstName, first.LastName)
	}
	if first.DateOfBirth == nil || first.DateOfBirth.Format("2006-01-02") != "1987-01-23" {
		t.Errorf("unexpected DOB: %v", first.DateOfBirth)
	}
	if first.MembershipValidTo == nil || first.MembershipValidTo.Format("2006-01-02") != "2026-06-13" {
		t.Errorf("unexpected membership valid to: %v", first.MembershipValidTo)
	}

	// Test second member (missing optional fields)
	second := members[1]
	if second.DateOfBirth != nil {
		t.Errorf("expected nil DOB, got %v", second.DateOfBirth)
	}
	if second.MembershipValidFrom != nil {
		t.Errorf("expected nil membership valid from, got %v", second.MembershipValidFrom)
	}

	// Test String() output
	str := first.String()
	if !strings.Contains(str, "Alice") || !strings.Contains(str, "Test") {
		t.Errorf("String() output does not contain expected names: %s", str)
	}
	if !strings.Contains(str, "1987-01-23") {
		t.Errorf("String() output missing DOB: %s", str)
	}
}

func TestParseMembersCSV_EmptyCSV(t *testing.T) {
	emptyCSV := []byte("")
	_, err := parseMembersCSV(emptyCSV)
	if err == nil {
		t.Fatal("expected error for empty CSV, got nil")
	}
}

func TestParseMembersCSV_InvalidDate(t *testing.T) {
	csvData := []byte(`"First Name","Last Name","Date of Birth"
Test,User,not-a-date`)

	members, err := parseMembersCSV(csvData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if members[0].DateOfBirth != nil {
		t.Errorf("expected nil DateOfBirth for invalid date, got %v", members[0].DateOfBirth)
	}
}
