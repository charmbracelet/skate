package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dgraph-io/badger/v4"
)

func TestOpenKVOptions(t *testing.T) {
	tmpDir := t.TempDir()
	origStore := storePath
	storePath = tmpDir
	defer func() { storePath = origStore }()

	db, err := openKV("testdb")
	if err != nil {
		t.Fatalf("expected openKV to succeed, got %v", err)
	}
	defer db.Close()

	opts := db.Opts()
	expectedVlogSize := int64(64 << 20)
	if opts.ValueLogFileSize != expectedVlogSize {
		t.Errorf("expected ValueLogFileSize to be %d, got %d", expectedVlogSize, opts.ValueLogFileSize)
	}
	if !opts.CompactL0OnClose {
		t.Errorf("expected CompactL0OnClose to be true, got %v", opts.CompactL0OnClose)
	}
}

func TestSetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	origStore := storePath
	storePath = tmpDir
	defer func() { storePath = origStore }()

	db, err := openKV("testdb")
	if err != nil {
		t.Fatalf("openKV failed: %v", err)
	}
	defer db.Close()

	key := []byte("hello")
	expectedValue := []byte("world")

	err = wrap(db, false, func(tx *badger.Txn) error {
		return tx.Set(key, expectedValue)
	})
	if err != nil {
		t.Fatalf("wrap set failed: %v", err)
	}

	var gotValue []byte
	err = wrap(db, true, func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return err
		}
		gotValue, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		t.Fatalf("wrap get failed: %v", err)
	}

	if !bytes.Equal(gotValue, expectedValue) {
		t.Fatalf("expected value %q, got %q", expectedValue, gotValue)
	}
}

func TestDelete(t *testing.T) {
	tmpDir := t.TempDir()
	origStore := storePath
	storePath = tmpDir
	defer func() { storePath = origStore }()

	db, err := openKV("testdb")
	if err != nil {
		t.Fatalf("openKV failed: %v", err)
	}
	defer db.Close()

	key := []byte("todelete")
	val := []byte("value")

	err = wrap(db, false, func(tx *badger.Txn) error {
		return tx.Set(key, val)
	})
	if err != nil {
		t.Fatalf("wrap set failed: %v", err)
	}

	err = wrap(db, false, func(tx *badger.Txn) error {
		return tx.Delete(key)
	})
	if err != nil {
		t.Fatalf("wrap delete failed: %v", err)
	}

	err = wrap(db, true, func(tx *badger.Txn) error {
		_, err := tx.Get(key)
		return err
	})
	if !errors.Is(err, badger.ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestLargeValue(t *testing.T) {
	tmpDir := t.TempDir()
	origStore := storePath
	storePath = tmpDir
	defer func() { storePath = origStore }()

	db, err := openKV("testdb")
	if err != nil {
		t.Fatalf("openKV failed: %v", err)
	}

	key := []byte("largekey")
	// 2MB value exceeds default ValueThreshold (1MB) and exercises value log
	largeValue := bytes.Repeat([]byte("A"), 2<<20)

	err = wrap(db, false, func(tx *badger.Txn) error {
		return tx.Set(key, largeValue)
	})
	if err != nil {
		t.Fatalf("wrap set large value failed: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("db close failed: %v", err)
	}

	// Reopen to verify persistence and retrieval after compaction
	db2, err := openKV("testdb")
	if err != nil {
		t.Fatalf("openKV reopen failed: %v", err)
	}
	defer db2.Close()

	var gotValue []byte
	err = wrap(db2, true, func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			return err
		}
		gotValue, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		t.Fatalf("wrap get large value failed: %v", err)
	}

	if !bytes.Equal(gotValue, largeValue) {
		t.Fatalf("retrieved large value length %d does not match expected %d", len(gotValue), len(largeValue))
	}
}

func TestKeyParser(t *testing.T) {
	tests := []struct {
		input       string
		expectedKey string
		expectedDB  string
		expectErr   bool
	}{
		{input: "alpha", expectedKey: "alpha", expectedDB: "", expectErr: false},
		{input: "ALPHA", expectedKey: "alpha", expectedDB: "", expectErr: false},
		{input: "alpha@beta", expectedKey: "alpha", expectedDB: "beta", expectErr: false},
		{input: "ALPHA@BETA", expectedKey: "alpha", expectedDB: "beta", expectErr: false},
		{input: "alpha@beta@gamma", expectedKey: "", expectedDB: "", expectErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			k, db, err := keyParser(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Errorf("expected error for %q, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}
			if string(k) != tc.expectedKey {
				t.Errorf("expected key %q, got %q", tc.expectedKey, string(k))
			}
			if db != tc.expectedDB {
				t.Errorf("expected db %q, got %q", tc.expectedDB, db)
			}
		})
	}
}

func TestCompactionOnClose(t *testing.T) {
	tmpDir := t.TempDir()
	origStore := storePath
	storePath = tmpDir
	defer func() { storePath = origStore }()

	db, err := openKV("testdb")
	if err != nil {
		t.Fatalf("openKV failed: %v", err)
	}

	for i := 0; i < 100; i++ {
		key := []byte(filepath.Join("key", string(rune(i))))
		val := []byte("val")
		if err := wrap(db, false, func(tx *badger.Txn) error {
			return tx.Set(key, val)
		}); err != nil {
			t.Fatalf("set failed at %d: %v", i, err)
		}
	}

	for i := 0; i < 50; i++ {
		key := []byte(filepath.Join("key", string(rune(i))))
		if err := wrap(db, false, func(tx *badger.Txn) error {
			return tx.Delete(key)
		}); err != nil {
			t.Fatalf("delete failed at %d: %v", i, err)
		}
	}

	// Close database with CompactL0OnClose enabled
	if err := db.Close(); err != nil {
		t.Fatalf("db close failed: %v", err)
	}

	// Reopen and verify remaining keys are still accessible
	db2, err := openKV("testdb")
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer db2.Close()

	for i := 50; i < 100; i++ {
		key := []byte(filepath.Join("key", string(rune(i))))
		err := wrap(db2, true, func(tx *badger.Txn) error {
			_, err := tx.Get(key)
			return err
		})
		if err != nil {
			t.Errorf("expected key %d to exist: %v", i, err)
		}
	}
}

func TestGetFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	origStore := storePath
	storePath = tmpDir
	defer func() { storePath = origStore }()

	fp, err := getFilePath("mydb")
	if err != nil {
		t.Fatalf("getFilePath failed: %v", err)
	}
	expected := filepath.Join(tmpDir, "mydb")
	if fp != expected {
		t.Errorf("expected file path %q, got %q", expected, fp)
	}

	info, err := os.Stat(tmpDir)
	if err != nil || !info.IsDir() {
		t.Errorf("expected directory to be created, stat err: %v", err)
	}
}
