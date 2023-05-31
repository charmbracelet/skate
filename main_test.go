package main

import (
	"errors"
	"testing"
)

func TestFindDbs(t *testing.T) {
	defaultDbs := []string{
		"@spongebob",
		"@charm.sh.kv.user.default",
		"@charm.sh.skate.default",
	}
	tests := []struct {
		name string
		dbs  []string
		err  error
	}{
		{
			name: "@spon",
			dbs:  defaultDbs,
			err: suggestionNotFoundErr{
				suggestions: []string{
					"@spongebob",
				},
			},
		},
		{
			name: "@char",
			dbs:  defaultDbs,
			err: suggestionNotFoundErr{
				suggestions: []string{
					"@charm.sh.kv.user.default",
					"@charm.sh.skate.default",
				},
			},
		},
		{
			name: "spon",
			dbs:  defaultDbs,
			err: suggestionNotFoundErr{
				suggestions: []string{
					"@spongebob",
				},
			},
		},
		{
			name: "",
			dbs:  defaultDbs,
			err: suggestionNotFoundErr{
				suggestions: nil,
			},
		},
		{
			name: "endo",
			dbs:  defaultDbs,
			err: suggestionNotFoundErr{
				suggestions: nil,
			},
		},
		{
			name: "@endo",
			dbs:  defaultDbs,
			err: suggestionNotFoundErr{
				suggestions: nil,
			},
		},
		{
			name: "@spongebob",
			dbs:  defaultDbs,
			err:  nil,
		},
	}

	for _, tc := range tests {
		_, err := findDb(tc.name, tc.dbs)
		if tc.err != nil {
			if err == nil {
				t.Fatalf("expected an error, got: %v", err)
			}
			var perr suggestionNotFoundErr
			if !errors.As(err, &perr) {
				t.Fatalf("something went wrong! got: %v", err)
			}
			if len(err.(suggestionNotFoundErr).suggestions) !=
				len(tc.err.(suggestionNotFoundErr).suggestions) {
				t.Fatalf("got != want. got: %v, want: %v", err, tc.err)
			}
		}
		if err != nil && tc.err == nil {
			t.Fatalf("got an unexpected error: %v", err)
		}
	}
}
