package shared

import "github.com/cli/cli/v2/pkg/cmdutil"

// Autolink represents a repository autolink reference.
type Autolink struct {
	ID             int    `json:"id"`
	IsAlphanumeric bool   `json:"is_alphanumeric"`
	KeyPrefix      string `json:"key_prefix"`
	URLTemplate    string `json:"url_template"`
}

// AutolinkFields defines the set of fields available for autolink export.
var AutolinkFields = []string{
	"id",
	"isAlphanumeric",
	"keyPrefix",
	"urlTemplate",
}

// ExportData returns a map of the requested fields for JSON export.
func (a *Autolink) ExportData(fields []string) map[string]interface{} {
	return cmdutil.StructExportData(a, fields)
}
