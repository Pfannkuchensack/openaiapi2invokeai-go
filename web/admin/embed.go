package admin

import "embed"

//go:embed *.css *.js
var Static embed.FS
