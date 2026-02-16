package featuredetection_test

import (
	"os"
	"path/filepath"
	"testing"

	featuredetectionlint "github.com/cli/cli/v2/pkg/linter/featuredetection"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	// Override the package path to use our test stub instead of the real
	// internal/featuredetection package which can't be imported from testdata.
	original := featuredetectionlint.FeaturedetectionPkgPath
	featuredetectionlint.FeaturedetectionPkgPath = "featuredetection_stub"
	t.Cleanup(func() { featuredetectionlint.FeaturedetectionPkgPath = original })

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	testdata := filepath.Join(dir, "testdata")
	analysistest.Run(t, testdata, featuredetectionlint.Analyzer, "example")
}
