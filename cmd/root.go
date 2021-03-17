package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"

	"github.com/aybabtme/uniplot/histogram"
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
	fset := token.NewFileSet() // positions are relative to fset
	pkgs, err := parser.ParseDir(fset, pkg, dirFilter, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		publicFunctions := 0
		fileCount := 0
		data := []float64{}

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
		}
		hist := histogram.Hist(3, data)
		err := histogram.Fprint(os.Stdout, hist, histogram.Linear(20))
		if err != nil {
			return err
		}

		fmt.Printf("Package '%s' has %d exported function(s) across %d file(s)\n", pkg.Name, publicFunctions, fileCount)
	}

	return nil
}

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
