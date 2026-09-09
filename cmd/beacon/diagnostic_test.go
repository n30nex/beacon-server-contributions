package main

import "testing"

func TestDiagnosticQueryName(t *testing.T) {
	for _, tc := range []struct{ sql, name string }{
		{"-- name: UpsertPacket :one\nINSERT ...", "UpsertPacket"},
		{"SELECT 'sensitive-value'", "unnamed"},
		{"-- name: unsafe/value :one", "unnamed"},
		{"", "unnamed"},
	} {
		if got := diagnosticQueryName(tc.sql); got != tc.name {
			t.Fatalf("query label = %q, want %q", got, tc.name)
		}
	}
}
