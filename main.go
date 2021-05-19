package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
	"github.com/dgraph-io/badger/v3"
	"github.com/spf13/cobra"
)

var (
	Version   = ""
	CommitSHA = ""

	reverseIterate   bool
	keysIterate      bool
	valuesIterate    bool
	delimiterIterate string

	rootCmd = &cobra.Command{
		Use:    "",
		Hidden: false,
		Short:  "Skate, a personal key value store.",
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	setCmd = &cobra.Command{
		Use:    "set KEY[@DB] VALUE",
		Hidden: false,
		Short:  "Set a value for a key with an optional @ db.",
		Args:   cobra.MaximumNArgs(2),
		RunE:   set,
	}

	getCmd = &cobra.Command{
		Use:    "get KEY[@DB]",
		Hidden: false,
		Short:  "Get a value for a key with an optional @ db.",
		Args:   cobra.ExactArgs(1),
		RunE:   get,
	}

	deleteCmd = &cobra.Command{
		Use:    "delete KEY[@DB]",
		Hidden: false,
		Short:  "Delete a key with an optional @ db.",
		Args:   cobra.ExactArgs(1),
		RunE:   delete,
	}

	iterateCmd = &cobra.Command{
		Use:    "iterate [@DB]",
		Hidden: false,
		Short:  "Iterate over all key value pairs with an optional @ db.",
		Args:   cobra.MaximumNArgs(1),
		RunE:   iterate,
	}

	syncCmd = &cobra.Command{
		Use:    "sync [@DB]",
		Hidden: false,
		Short:  "Sync local db with latest Charm Cloud db.",
		Args:   cobra.MaximumNArgs(1),
		RunE:   sync,
	}

	resetCmd = &cobra.Command{
		Use:    "reset [@DB]",
		Hidden: false,
		Short:  "Delete local db and pull down fresh copy from Charm Cloud.",
		Args:   cobra.MaximumNArgs(1),
		RunE:   reset,
	}
)

func set(cmd *cobra.Command, args []string) error {
	k, n, err := keyParser(args[0])
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	if len(args) == 2 {
		return db.Set(k, []byte(args[1]))
	}
	return db.SetReader(k, os.Stdin)
}

func get(cmd *cobra.Command, args []string) error {
	k, n, err := keyParser(args[0])
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	v, err := db.Get(k)
	if err != nil {
		return err
	}
	fmt.Println(string(v))
	return nil
}

func delete(cmd *cobra.Command, args []string) error {
	k, n, err := keyParser(args[0])
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	return db.Delete(k)
}

func iterate(cmd *cobra.Command, args []string) error {
	var k string
	var pf string
	if keysIterate || valuesIterate {
		pf = "%s\n"
	} else {
		pf = fmt.Sprintf("%%s%s%%s\n", delimiterIterate)
	}
	if len(args) == 1 {
		k = args[0]
	}
	_, n, err := keyParser(k)
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	db.Sync()
	return db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		opts.Reverse = reverseIterate
		if keysIterate {
			opts.PrefetchValues = false
		}
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if keysIterate {
				fmt.Printf(pf, k)
				continue
			}
			err := item.Value(func(v []byte) error {
				if valuesIterate {
					fmt.Printf(pf, v)
				} else {
					fmt.Printf(pf, k, v)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func sync(cmd *cobra.Command, args []string) error {
	n, err := nameFromArgs(args)
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	return db.Sync()
}

func reset(cmd *cobra.Command, args []string) error {
	n, err := nameFromArgs(args)
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	return db.Reset()
}

func nameFromArgs(args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	_, n, err := keyParser(args[0])
	if err != nil {
		return "", err
	}
	return n, nil
}

func keyParser(k string) ([]byte, string, error) {
	var key, db string
	ps := strings.Split(k, "@")
	switch len(ps) {
	case 1:
		key = strings.ToLower(ps[0])
	case 2:
		key = strings.ToLower(ps[0])
		db = strings.ToLower(ps[1])
	default:
		return nil, "", fmt.Errorf("bad key format, use KEY@DB")
	}
	return []byte(key), db, nil
}

func openKV(name string) (*kv.KV, error) {
	dd, err := client.DataPath()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = "charm.sh..user.default"
	}
	return kv.OpenWithDefaults(name, fmt.Sprintf("%s/", dd))
}

func init() {
	if len(CommitSHA) >= 7 {
		vt := rootCmd.VersionTemplate()
		rootCmd.SetVersionTemplate(vt[:len(vt)-1] + " (" + CommitSHA[0:7] + ")\n")
	}
	if Version == "" {
		Version = "unknown (built from source)"
	}
	rootCmd.Version = Version

	iterateCmd.Flags().BoolVarP(&reverseIterate, "reverse", "r", false, "iterate in reverse lexicographic order")
	iterateCmd.Flags().BoolVarP(&keysIterate, "keys-only", "k", false, "only print keys and don't fetch values from the db")
	iterateCmd.Flags().BoolVarP(&valuesIterate, "values-only", "v", false, "only print values")
	iterateCmd.Flags().StringVarP(&delimiterIterate, "delimiter", "d", "\t", "delimiter to separate keys and values")

	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(iterateCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(resetCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
