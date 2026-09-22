// Package contents embeds the authored learning content that ships with the API.
package contents

import "embed"

// Grammar holds one JSON file per grammar topic under grammar/.
//
//go:embed grammar/*.json
var Grammar embed.FS

// GrammarDir is the directory inside Grammar that holds the topic files.
const GrammarDir = "grammar"

// Discussions holds one JSON file per conversation question set under
// discussions/.
//
//go:embed discussions/*.json
var Discussions embed.FS

// DiscussionsDir is the directory inside Discussions that holds the set files.
const DiscussionsDir = "discussions"
