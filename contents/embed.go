// Package contents embeds the authored learning content that ships with the API.
package contents

import "embed"

// Grammar holds one JSON file per grammar topic under grammar/.
//
//go:embed grammar/*.json
var Grammar embed.FS

// GrammarDir is the directory inside Grammar that holds the topic files.
const GrammarDir = "grammar"
