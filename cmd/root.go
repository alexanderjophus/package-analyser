package cmd

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/aybabtme/uniplot/histogram"
	"github.com/google/go-github/v33/github"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "package-analyser",
	Short: "Analyses packages to give a 100ft view of how they look",
	Run: func(cmd *cobra.Command, args []string) {
		if err := run(args[0]); err != nil {
			log.Fatal(err)
		}
	},
}

func dirFilter(f fs.FileInfo) bool { return true }

func run(pkg string) error {
	if strings.HasPrefix(pkg, "github.com") {
		return parseGithubPackage(pkg)
	}

	return parseLocalPackage(pkg)
}

func parseGithubPackage(pkg string) error {
	u, err := url.Parse(pkg)
	if err != nil {
		return fmt.Errorf("parsing url: %w", err)
	}

	s := strings.Split(u.Path, "/")
	if len(s) < 3 {
		return fmt.Errorf("package not specified")
	}
	path := strings.Join(s[3:], "/")

	client := github.NewClient(nil)

	_, dirC, _, err := client.Repositories.GetContents(context.Background(), s[1], s[2], path, nil)
	if err != nil {
		return fmt.Errorf("getting package: %w", err)
	}

	publicFunctions := 0
	fileCount := 0
	data := []float64{}
	imports := make(map[string]bool)

	for _, f := range dirC {
		if !strings.HasSuffix(f.GetName(), ".go") {
			continue
		}
		fileCount++
		fileC, _, _, err := client.Repositories.GetContents(context.Background(), s[1], s[2], f.GetPath(), nil)
		if err != nil {
			return fmt.Errorf("getting file: %w", err)
		}
		c, err := fileC.GetContent()
		if err != nil {
			return fmt.Errorf("getting file contents: %w", err)
		}

		fset := token.NewFileSet() // positions are relative to fset

		fp, err := parser.ParseFile(fset, f.GetName(), c, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parsing file: %w", err)
		}

		publicFuncsPerFile := 0.
		for _, d := range fp.Decls {
			if fn, isFn := d.(*ast.FuncDecl); isFn && ast.IsExported(fn.Name.Name) {
				publicFunctions++
				publicFuncsPerFile++
			}
		}
		data = append(data, publicFuncsPerFile)

		for _, i := range fp.Imports {
			imports[i.Path.Value] = true
		}
	}
	hist := histogram.Hist(5, data)
	err = histogram.Fprint(os.Stdout, hist, histogram.Linear(20))
	if err != nil {
		return err
	}

	fmt.Printf("Package has %d exported function(s) across %d file(s)\n", publicFunctions, fileCount)

	i := alphabetical(toSlice(imports))
	sort.Sort(i)
	fmt.Printf("Importing the following: %v\n", i)

	return nil
}

func parseLocalPackage(pkg string) error {
	fset := token.NewFileSet() // positions are relative to fset
	pkgs, err := parser.ParseDir(fset, pkg, dirFilter, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		publicFunctions := 0
		fileCount := 0
		data := []float64{}
		imports := make(map[string]bool)

		for _, f := range pkg.Files {
			fileCount++
			publicFuncsPerFile := 0.
			for _, d := range f.Decls {
				if fn, isFn := d.(*ast.FuncDecl); isFn && ast.IsExported(fn.Name.Name) {
					publicFunctions++
					publicFuncsPerFile++
				}
			}
			data = append(data, publicFuncsPerFile)

			for _, i := range f.Imports {
				imports[i.Path.Value] = true
			}
		}
		hist := histogram.Hist(5, data)
		err := histogram.Fprint(os.Stdout, hist, histogram.Linear(20))
		if err != nil {
			return err
		}

		fmt.Printf("Package '%s' has %d exported function(s) across %d file(s)\n", pkg.Name, publicFunctions, fileCount)

		i := alphabetical(toSlice(imports))
		sort.Sort(i)
		fmt.Printf("Importing the following: %v\n", i)
	}

	return nil
}

func toSlice(is map[string]bool) []string {
	out := []string{}
	for i := range is {
		out = append(out, i)
	}
	return out
}

type alphabetical []string

func (a alphabetical) Len() int           { return len(a) }
func (a alphabetical) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a alphabetical) Less(i, j int) bool { return a[i] < a[j] }

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
