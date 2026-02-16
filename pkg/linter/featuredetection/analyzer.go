// Package featuredetection provides an analysis.Analyzer that checks for
// required TODO comments near if-statements that reference feature detection
// struct fields from internal/featuredetection.
//
// Feature detection is used to branch behavior based on GHES vs github.com
// capabilities. When GHES eventually adds support for a feature, the detection
// code becomes dead code. The TODO comment provides a greppable identifier so
// that all related code can be found and cleaned up when the time comes.
package featuredetection

import (
	"go/ast"
	"go/types"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// FeaturedetectionPkgPath is the import path of the featuredetection package.
// It is a variable so tests can override it with a stub package path.
var FeaturedetectionPkgPath = "github.com/cli/cli/v2/internal/featuredetection"

// featureStructs is the set of struct type names from the featuredetection
// package whose boolean fields trigger the TODO requirement.
var featureStructs = map[string]bool{
	"IssueFeatures":       true,
	"PullRequestFeatures": true,
	"RepositoryFeatures":  true,
	"SearchFeatures":      true,
	"ReleaseFeatures":     true,
}

// todoPattern matches "// TODO <identifier>" or "// TODO: <identifier>" style comments.
var todoPattern = regexp.MustCompile(`//\s*TODO[\s:]+\S+`)

var Analyzer = &analysis.Analyzer{
	Name:     "featuredetection",
	Doc:      "checks that if-statements referencing featuredetection fields have a TODO comment within 10 lines above",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.IfStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		ifStmt := n.(*ast.IfStmt)

		fieldName := referencesFeatureField(pass, ifStmt.Cond)
		if fieldName == "" {
			return
		}

		ifLine := pass.Fset.Position(ifStmt.Pos()).Line

		if hasTODOComment(pass, ifLine) {
			return
		}

		pass.Reportf(ifStmt.Pos(),
			"if-statement references featuredetection field %q but is missing a required TODO comment (e.g. // TODO <identifier>) within the 10 lines above",
			fieldName,
		)
	})

	return nil, nil
}

// referencesFeatureField walks the expression tree and returns the name of the
// first featuredetection struct field it finds, or "" if none.
func referencesFeatureField(pass *analysis.Pass, expr ast.Expr) string {
	var found string
	ast.Inspect(expr, func(n ast.Node) bool {
		if found != "" {
			return false
		}
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		selObj := pass.TypesInfo.Selections[sel]
		if selObj == nil {
			return true
		}

		recv := selObj.Recv()
		// Dereference pointer types.
		if ptr, ok := recv.(*types.Pointer); ok {
			recv = ptr.Elem()
		}
		named, ok := recv.(*types.Named)
		if !ok {
			return true
		}

		obj := named.Obj()
		if obj.Pkg() == nil || obj.Pkg().Path() != FeaturedetectionPkgPath {
			return true
		}
		if !featureStructs[obj.Name()] {
			return true
		}

		found = sel.Sel.Name
		return false
	})
	return found
}

// hasTODOComment checks whether any comment in the 10 lines preceding ifLine
// matches the TODO pattern.
func hasTODOComment(pass *analysis.Pass, ifLine int) bool {
	for _, f := range pass.Files {
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				commentLine := pass.Fset.Position(c.Pos()).Line
				if commentLine >= ifLine-10 && commentLine < ifLine {
					if isTODOComment(c.Text) {
						return true
					}
				}
			}
		}
	}
	return false
}

// isTODOComment returns true if the comment text matches the required TODO format:
// "// TODO <identifier>" or "// TODO: <identifier>"
func isTODOComment(text string) bool {
	if !strings.Contains(text, "TODO") {
		return false
	}
	return todoPattern.MatchString(text)
}
