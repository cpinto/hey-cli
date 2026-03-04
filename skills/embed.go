// Package skills embeds the skill files in the binary.
package skills

import "embed"

//go:embed hey
var FS embed.FS
