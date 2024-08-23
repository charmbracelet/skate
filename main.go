package main

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/agnivade/levenshtein"
	"github.com/charmbracelet/lipgloss"
	"github.com/dgraph-io/badger/v4"
	gap "github.com/muesli/go-app-paths"
	mcobra "github.com/muesli/mango-cobra"
	"github.com/muesli/roff"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Version set by goreleaser.
	Version = ""

	// CommitSHA set by goreleaser.
	CommitSHA = ""

	reverseIterate   bool
	keysIterate      bool
	valuesIterate    bool
	showBinary       bool
	delimiterIterate string

	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("204")).Bold(true)

	rootCmd = &cobra.Command{
		Use:   "skate",
		Short: "Skate, a personal key value store.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
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
		RunE:  del,
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

	manCmd = &cobra.Command{
		Use:    "man",
		Short:  "Generate man pages",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(*cobra.Command, []string) error {
			manPage, err := mcobra.NewManPage(1, rootCmd) //.
			if err != nil {
				return err
			}
			manPage = manPage.WithSection("Copyright", "(C) 2021-2024 Charmbracelet, Inc.\n"+
				"Released under MIT license.")
			fmt.Println(manPage.Build(roff.NewDocument()))
			return nil
		},
	}
)

type errDBNotFound struct {
	suggestions []string
}

func (err errDBNotFound) Error() string {
	if len(err.suggestions) == 0 {
		return "no suggestions found"
	}
	return fmt.Sprintf("did you mean %q", strings.Join(err.suggestions, ", "))
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
	defer db.Close() //nolint:errcheck
	if len(args) == 2 {
		return wrap(db, false, func(tx *badger.Txn) error {
			return tx.Set(k, []byte(args[1]))
		})
	}
	bts, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return err
	}
	return wrap(db, false, func(tx *badger.Txn) error {
		return tx.Set(k, bts)
	})
}

func get(_ *cobra.Command, args []string) error {
	k, n, err := keyParser(args[0])
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck
	var v []byte
	if err := wrap(db, true, func(tx *badger.Txn) error {
		item, err := tx.Get(k)
		if err != nil {
			return err
		}
		v, err = item.ValueCopy(nil)
		return err
	}); err != nil {
		return err
	}
	printFromKV("%s", v)
	return nil
}

func del(_ *cobra.Command, args []string) error {
	k, n, err := keyParser(args[0])
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck

	return wrap(db, false, func(tx *badger.Txn) error {
		return tx.Delete(k)
	})
}

// TODO: use lists/tables/trees for this?
func listDbs(*cobra.Command, []string) error {
	dbs, err := getDbs()
	for _, db := range dbs {
		fmt.Println(db)
	}
	return err
}

// getDbs: returns a formatted list of available Skate DBs.
func getDbs() ([]string, error) {
	filepath, err := getFilePath()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath)
	if err != nil {
		return nil, err
	}
	var dbList []string
	for _, e := range entries {
		if e.IsDir() {
			dbList = append(dbList, e.Name())
		}
	}
	return formatDbs(dbList), nil
}

func formatDbs(dbs []string) []string {
	out := make([]string, 0, len(dbs))
	for _, db := range dbs {
		out = append(out, "@"+db)
	}
	return out
}

// getFilePath: get the file path to the skate databases.
func getFilePath(args ...string) (string, error) {
	scope := gap.NewScope(gap.User, "charm")
	dd, pathErr := scope.DataPath("")
	if pathErr != nil {
		return "", pathErr
	}
	dir := filepath.Join(dd, "kv")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(append([]string{dir}, args...)...), nil
}

// deleteDb: delete a Skate database.
func deleteDb(_ *cobra.Command, args []string) error {
	path, err := findDb(args[0])
	var errNotFound errDBNotFound
	if errors.As(err, &errNotFound) {
		fmt.Fprintf(os.Stderr, "%q does not exist, %s\n", args[0], err.Error())
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %s", err.Error())
		os.Exit(1)
	}
	var confirmation string

	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		path = filepath.Join("~", strings.TrimPrefix(path, home))
	}
	message := fmt.Sprintf("Are you sure you want to delete '%s' and all its contents? (y/n)", warningStyle.Render(path))
	message = lipgloss.NewStyle().Width(78).Render(message)
	fmt.Println(message)

	// TODO: use huh
	if _, err := fmt.Scanln(&confirmation); err != nil {
		return err
	}
	if confirmation == "y" {
		return os.RemoveAll(path)
	}
	fmt.Fprintf(os.Stderr, "Did not delete %q\n", path)
	return nil
}

// findDb: returns the path to the named db or an errDBNotFound if no
// match is found.
func findDb(name string) (string, error) {
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
			diff := int(math.Abs(float64(len(db) - len(name))))
			levenshteinDistance := levenshtein.ComputeDistance(name, db)
			suggestByLevenshtein := levenshteinDistance <= diff
			if suggestByLevenshtein {
				suggestions = append(suggestions, db)
			}
		}
		return "", errDBNotFound{suggestions: suggestions}
	}
	return path, nil
}

func list(_ *cobra.Command, args []string) error {
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
		defer it.Close() //nolint:errcheck
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
	isatty := term.IsTerminal(int(os.Stdin.Fd()))
	for _, v := range vs {
		if isatty && !showBinary && !utf8.Valid(v) {
			fvs = append(fvs, nb)
		} else {
			fvs = append(fvs, string(v))
		}
	}
	fmt.Printf(pf, fvs...)
	if isatty && !strings.HasSuffix(pf, "\n") {
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

func openKV(name string) (*badger.DB, error) {
	if name == "" {
		name = "default"
	}
	path, err := getFilePath(name)
	if err != nil {
		return nil, err
	}
	return badger.Open(badger.DefaultOptions(path).WithLoggingLevel(badger.ERROR))
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
		manCmd,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func wrap(db *badger.DB, readonly bool, fn func(tx *badger.Txn) error) error {
	tx := db.NewTransaction(!readonly)
	if err := fn(tx); err != nil {
		tx.Discard()
		return err
	}
	return tx.Commit() //nolint:wrapcheck
}
