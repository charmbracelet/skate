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
			err: dbNotFoundErr{
				name: "@spon",
				suggestions: []string{
					"@spongebob",
				},
				isEmpty: false,
			},
		},
		{
			name: "spon",
			dbs:  defaultDbs,
			err: dbNotFoundErr{
				name:        "spon",
				suggestions: defaultDbs,
				isEmpty:     false,
			},
		},
		{
			name: "",
			dbs:  defaultDbs,
			err: dbNotFoundErr{
				name:        "spon",
				suggestions: defaultDbs,
				isEmpty:     true,
			},
		},
		{
			name: "endo",
			dbs:  defaultDbs,
			err: dbNotFoundErr{
				name:        "spon",
				suggestions: defaultDbs,
				isEmpty:     true,
			},
		},
		{
			name: "@endo",
			dbs:  defaultDbs,
			err: dbNotFoundErr{
				name:        "spon",
				suggestions: defaultDbs,
				isEmpty:     true,
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
			if err == nil || errors.Is(err, dbNotFoundErr{}) {
				t.Fatalf("expected an error, got: %v", err)
			}
			gIsEmpty := err.(dbNotFoundErr).isEmpty
			wIsEmpty := tc.err.(dbNotFoundErr).isEmpty
			if gIsEmpty != wIsEmpty {
				t.Fatalf("got: %t, want: %t", gIsEmpty, wIsEmpty)
			}
		}
		if err != nil && tc.err == nil {
			t.Fatalf("got an unexpected error: %v", err)
		}
	}
}
