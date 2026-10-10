// Package jsonout renders an Analysis as stable, versioned JSON. The schema
// is documented in docs/json-schema.md.
package jsonout

import (
	"encoding/json"
	"io"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
)

// Render writes the analysis as indented JSON. Map keys are sorted by
// encoding/json, so output is deterministic.
func Render(w io.Writer, a *analyze.Analysis) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(a)
}
