//go:build embed_web

package webui

import (
	"embed"
	"io/fs"
)

//go:embed dist
var embeddedDist embed.FS

func Embedded() bool {
	return true
}

func Dist() (fs.FS, error) {
	return fs.Sub(embeddedDist, "dist")
}
