package rag

import "testing"

func TestSanitizeSegmentRejectsTraversal(t *testing.T) {
	for _, bad := range []string{"../x", "a/b", `a\b`, "", "..", "foo:bar"} {
		if _, err := SanitizeSegment(bad); err == nil {
			t.Fatalf("expected reject %q", bad)
		}
	}
	if _, err := SanitizeSegment("code-style"); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertDocumentRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	idx := NewIndex(dir)
	_ = idx.Load()
	_, err := idx.UpsertDocument("../evil", "t", "general", "body", nil)
	if err == nil {
		t.Fatal("expected path rejection")
	}
	_, err = idx.UpsertDocument("ok", "t", "../evil", "body", nil)
	if err == nil {
		t.Fatal("expected category rejection")
	}
}
