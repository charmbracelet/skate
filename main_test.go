package main

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestFindDbs(t *testing.T) {
	defaultDbs := []string{
		"spongebob",
		"charm.sh.kv.user.default",
		"charm.sh.skate.default",
	}
	tests := []struct {
		name string
		dbs  []string
		err  error
	}{
		{
			name: "@spon",
			dbs:  defaultDbs,
			err: errDBNotFound{
				suggestions: []string{
					"@spongebob",
				},
			},
		},
		{
			name: "@char",
			dbs:  defaultDbs,
			err: errDBNotFound{
				suggestions: []string{
					"@charm.sh.kv.user.default",
					"@charm.sh.skate.default",
				},
			},
		},
		{
			name: "spon",
			dbs:  defaultDbs,
			err: errDBNotFound{
				suggestions: []string{
					"@spongebob",
				},
			},
		},
		{
			name: "",
			dbs:  defaultDbs,
			err: errDBNotFound{
				suggestions: nil,
			},
		},
		{
			name: "endo",
			dbs:  defaultDbs,
			err: errDBNotFound{
				suggestions: nil,
			},
		},
		{
			name: "@endo",
			dbs:  defaultDbs,
			err: errDBNotFound{
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
		t.Run(tc.name, func(t *testing.T) {
			path := setup(t, tc.dbs)
			defer teardown(path)
			_, err := findDb(tc.name, tc.dbs)
			if tc.err != nil {
				if err == nil {
					t.Fatalf("expected an error, got: %v", err)
				}
				var perr errDBNotFound
				if !errors.As(err, &perr) {
					t.Fatalf("something went wrong! got: %v", err)
				}
				if len(err.(errDBNotFound).suggestions) !=
					len(tc.err.(errDBNotFound).suggestions) {

					t.Fatalf("got != want. got: %v, want: %v", err, tc.err)
				}
			}
			if err != nil && tc.err == nil {
				t.Fatalf("got an unexpected error: %v", err)
			}
			if err := teardown(path); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TODO: use local charm server for testing instead of pinging cloud services
func setup(t *testing.T, dbs []string) string {
	// set up charm kv temp path for tests
	dir := os.TempDir()
	path := fmt.Sprintf("%scharm-tests", dir)
	t.Setenv("CHARM_DATA_DIR", path)
	// create the kv dbs
	for _, db := range dbs {
		charmKV, err := openKV(db)
		if err != nil {
			t.Fatal(err)
		}
		if err := charmKV.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func teardown(path string) error {
	return os.RemoveAll(path)
}