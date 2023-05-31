package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/agnivade/levenshtein"
	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/cmd"
	"github.com/charmbracelet/charm/kv"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/lipgloss"
	"github.com/dgraph-io/badger/v3"
	mcobra "github.com/muesli/mango-cobra"
	"github.com/muesli/roff"
	"github.com/spf13/cobra"
)

// distance: an arbitrary number to dictate suggestions
const distance = 3

var (
	Version   = ""
	CommitSHA = ""

	reverseIterate   bool
	keysIterate      bool
	valuesIterate    bool
	showBinary       bool
	delimiterIterate string

	warningStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FD5B5B")).Italic(true)
	highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5FD2"))

	rootCmd = &cobra.Command{
		Use:   "skate",
		Short: "Skate, a personal key value store.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	setCmd = &cobra.Command{
		Use:   "set KEY[@DB] VALUE",
		Short: "Set a value for a key with an optional @ db.",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  set,
	}

	getCmd = &cobra.Command{
		Use:           "get KEY[@DB]",
		Short:         "Get a value for a key with an optional @ db.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE:          get,
	}

	deleteCmd = &cobra.Command{
		Use:   "delete KEY[@DB]",
		Short: "Delete a key with an optional @ db.",
		Args:  cobra.ExactArgs(1),
		RunE:  delete,
	}

	listCmd = &cobra.Command{
		Use:   "list [@DB]",
		Short: "List key value pairs with an optional @ db.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  list,
	}

	listDbsCmd = &cobra.Command{
		Use:   "list-dbs",
		Short: "List databases.",
		Args:  cobra.NoArgs,
		RunE:  listDbs,
	}

	deleteDbCmd = &cobra.Command{
		Use:    "delete-db [@DB]",
		Hidden: false,
		Short:  "Delete a database",
		Args:   cobra.MinimumNArgs(1),
		RunE:   deleteDb,
	}

	syncCmd = &cobra.Command{
		Use:   "sync [@DB]",
		Short: "Sync local db with latest Charm Cloud db.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  sync,
	}

	resetCmd = &cobra.Command{
		Use:   "reset [@DB]",
		Short: "Delete local db and pull down fresh copy from Charm Cloud.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  reset,
	}

	manCmd = &cobra.Command{
		Use:    "man",
		Short:  "Generate man pages",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			manPage, err := mcobra.NewManPage(1, rootCmd) //.
			if err != nil {
				return err
			}

			manPage = manPage.WithSection("Copyright", "(C) 2021-2022 Charmbracelet, Inc.\n"+
				"Released under MIT license.")
			fmt.Println(manPage.Build(roff.NewDocument()))
			return nil
		},
	}
)

type suggestionNotFoundErr struct {
	suggestions []string
}

func (e suggestionNotFoundErr) Error() string {
	if len(e.suggestions) == 0 {
		return "no suggestions found"
	}
	return fmt.Sprintf("did you mean %q", strings.Join(e.suggestions, ", "))
}

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
	printFromKV("%s", v)
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

func listDbs(cmd *cobra.Command, args []string) error {
	dbs, err := getDbs()
	for _, db := range dbs {
		fmt.Println(db)
	}
	return err
}

// getDbs: returns a formatted list of available Skate DBs
func getDbs() ([]string, error) {
	filepath, err := getFilePath()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, "@"+e.Name())
		}
	}
	return out, nil
}

// getFilePath: get the file path to the skate databases.
func getFilePath(args ...string) (string, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return "", err
	}
	dd, err := cc.DataPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{dd, "kv"}, args...)...), err
}

// deleteDb: delete a Skate database.
func deleteDb(cmd *cobra.Command, args []string) error {
	dbs, err := getDbs()
	if err != nil {
		return err
	}
	path, err := findDb(args[0], dbs)
	if errors.Is(err, suggestionNotFoundErr{}) {
		fmt.Printf("%q does not exist, %s", args[0], err.Error())
		os.Exit(1)
	}
	var confirmation string
	fmt.Printf("are you sure you want to delete '%s' and all its contents?(y/n) ", warningStyle.Render(path))
	fmt.Scanln(&confirmation)
	if confirmation == "y" {
		return os.RemoveAll(path)
	}
	fmt.Printf("did not delete %q\n", path)
	return nil
}

// findDb: returns the path to the named db, if found.
func findDb(name string, dbs []string) (string, error) {
	sName, err := nameFromArgs([]string{name})
	if err != nil {
		return "", err
	}
	path, err := getFilePath(sName)
	if err != nil {
		return "", err
	}
	_, err = os.Stat(path)
	if sName == "" || os.IsNotExist(err) {
		dbs, err := getDbs()
		if err != nil {
			return "", err
		}
		var suggestions []string
		for _, db := range dbs {
			levenshteinDistance := levenshtein.ComputeDistance(name, db)
			suggestByLevenshtein := levenshteinDistance <= distance
			suggestByPrefix := strings.HasPrefix(name, db[:distance])
			suggestByPrefixAlt := strings.HasPrefix("@"+name, db[:distance])
			if suggestByLevenshtein || suggestByPrefix || suggestByPrefixAlt {
				suggestions = append(suggestions, db)
			}
		}
		return "", suggestionNotFoundErr{suggestions: suggestions}
	}
	return path, nil
}

func list(cmd *cobra.Command, args []string) error {
	var k string
	var pf string
	if keysIterate || valuesIterate {
		pf = "%s\n"
	} else {
		var err error
		pf, err = strconv.Unquote(fmt.Sprintf(`"%%s%s%%s\n"`, delimiterIterate))
		if err != nil {
			return err
		}
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
	err = db.Sync()
	if err != nil {
		return err
	}
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
				printFromKV(pf, k)
				continue
			}
			err := item.Value(func(v []byte) error {
				if valuesIterate {
					printFromKV(pf, v)
				} else {
					printFromKV(pf, k, v)
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

func printFromKV(pf string, vs ...[]byte) {
	nb := "(omitted binary data)"
	fvs := make([]interface{}, 0)
	for _, v := range vs {
		if common.IsTTY() && !showBinary && !utf8.Valid(v) {
			fvs = append(fvs, nb)
		} else {
			fvs = append(fvs, string(v))
		}
	}
	fmt.Printf(pf, fvs...)
	if common.IsTTY() && !strings.HasSuffix(pf, "\n") {
		fmt.Println()
	}
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
	if name == "" {
		name = "charm.sh.skate.default"
	}
	return kv.OpenWithDefaults(name)
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
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	listCmd.Flags().BoolVarP(&reverseIterate, "reverse", "r", false, "list in reverse lexicographic order")
	listCmd.Flags().BoolVarP(&keysIterate, "keys-only", "k", false, "only print keys and don't fetch values from the db")
	listCmd.Flags().BoolVarP(&valuesIterate, "values-only", "v", false, "only print values")
	listCmd.Flags().StringVarP(&delimiterIterate, "delimiter", "d", "\t", "delimiter to separate keys and values")
	listCmd.Flags().BoolVarP(&showBinary, "show-binary", "b", false, "print binary values")
	getCmd.Flags().BoolVarP(&showBinary, "show-binary", "b", false, "print binary values")

	rootCmd.AddCommand(
		getCmd,
		setCmd,
		deleteCmd,
		listCmd,
		listDbsCmd,
		deleteDbCmd,
		syncCmd,
		resetCmd,
		cmd.LinkCmd("skate"),
		manCmd,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
