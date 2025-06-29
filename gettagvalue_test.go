package driftflow

import "testing"

func TestGetTagValueTrimSpaces(t *testing.T) {
	tag := "column: display_name ;type: varchar(45)"
	if got := getTagValue(tag, "column"); got != "display_name" {
		t.Fatalf("column: got %q", got)
	}
	if got := getTagValue(tag, "type"); got != "varchar(45)" {
		t.Fatalf("type: got %q", got)
	}
}
