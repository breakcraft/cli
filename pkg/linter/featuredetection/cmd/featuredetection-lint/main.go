// Command featuredetection-lint runs the featuredetection analyzer.
// Usage: go vet -vettool=$(go build ./pkg/linter/featuredetection/cmd/featuredetection-lint) ./...
package main

import (
	featuredetection "github.com/cli/cli/v2/pkg/linter/featuredetection"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(featuredetection.Analyzer)
}
