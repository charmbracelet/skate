package main

import (
	"errors"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/charmbracelet/charm/testserver"
)

func TestFindDbs(t *testing.T) {
	defaultDbs := []string{
		"spongebob",
		"charm.sh.kv.user.default",
		"charm.sh.skate.default",
		"sk",
	}
	tests := []struct {
		tname string
		name  string
		dbs   []string
		err   error
	}{
		{
			tname: "unique, single char",
			name:  "p",
			dbs:   defaultDbs,
			err: errDBNotFound{
				suggestions: []string{
					"@spongebob",
				},
			},
		},
		{
			tname: "name > db",
			name:  "pcharm.sh.kv.user.defaultii",
			dbs:   defaultDbs,
			err: errDBNotFound{
				suggestions: nil,
			},
		},

		{
			tname: "empty",
			name:  "",
			dbs:   defaultDbs,
			err: errDBNotFound{
				suggestions: formatDbs(defaultDbs),
			},
		},
		{
			tname: "single match",
			name:  "@spon",
			dbs:   defaultDbs,
			err: errDBNotFound{
				suggestions: []string{
					"@spongebob",
				},
			},
		},
		{
			tname: "charm match",
			name:  "@char",
			dbs:   defaultDbs,
			err: errDBNotFound{
				suggestions: []string{
					"@charm.sh.kv.user.default",
					"@charm.sh.skate.default",
				},
			},
		},
		{
			tname: "single match, no @",
			name:  "spon",
			dbs:   defaultDbs,
			err: errDBNotFound{
				suggestions: []string{
					"@spongebob",
				},
			},
		},
		{
			tname: "no match, no @",
			name:  "endo",
			dbs:   defaultDbs,
			err: errDBNotFound{
				suggestions: nil,
			},
		},
		{
			tname: "no match",
			name:  "@endo",
			dbs:   defaultDbs,
			err: errDBNotFound{
				suggestions: nil,
			},
		},
		{
			tname: "exact match",
			name:  "@spongebob",
			dbs:   defaultDbs,
			err:   nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.tname, func(t *testing.T) {
			path := setup(t, tc.dbs)
			defer teardown(path)
			_, err := findDb(tc.name, tc.dbs)
			if tc.err != nil {
				if err == nil {
					t.Fatalf("expected an error, got: %v", err)
				}
				// check we got the right type of error
				var perr errDBNotFound
				if !errors.As(err, &perr) {
					t.Fatalf("something went wrong! got: %v", err)
				}
				// check suggestions match
				gotSuggestions := err.(errDBNotFound).suggestions
				wantSuggestions := tc.err.(errDBNotFound).suggestions
				sort.Strings(gotSuggestions)
				sort.Strings(wantSuggestions)
				if !reflect.DeepEqual(gotSuggestions, wantSuggestions) {
					t.Fatalf("got != want. got: %v, want: %v", err, tc.err)
				}
			}
			if err != nil && tc.err == nil {
				t.Fatalf("got an unexpected error: %v", err)
			}
		})
	}
}

func setup(t *testing.T, dbs []string) string {
	// set up a charm server for testing
	client := testserver.SetupTestServer(t)
	// add the skate dbs
	for _, db := range dbs {
		charmKV, err := openKV(db)
		if err != nil {
			t.Fatal(err)
		}
		if err := charmKV.Close(); err != nil {
			t.Fatal(err)
		}
	}
	path, err := client.DataPath()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func teardown(path string) error {
	return os.RemoveAll(path)
}
