package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
	"github.com/spf13/cobra"
)

var (
	Version   = ""
	CommitSHA = ""

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

	keysCmd = &cobra.Command{
		Use:    "keys [@DB]",
		Hidden: false,
		Short:  "List all keys with an optional @ db.",
		Args:   cobra.MaximumNArgs(1),
		RunE:   keys,
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

func keys(cmd *cobra.Command, args []string) error {
	var k string
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
	ks, err := db.Keys()
	if err != nil {
		panic(err)
	}
	for _, k := range ks {
		fmt.Println(string(k))
	}
	return nil
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
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(keysCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(resetCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
