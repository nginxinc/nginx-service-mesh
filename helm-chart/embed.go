// Package chart contains the helm charts embedded into a Go filesystem struct.
package chart

import "embed"

//go:embed *
//go:embed templates/_helpers.tpl
var helmFiles embed.FS

// HelmFiles returns the embedded helm files.
func HelmFiles() embed.FS {
	return helmFiles
}
