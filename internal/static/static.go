package static

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed public/**
var embeddedStatic embed.FS

func FS(useEmbedded bool) (fs.FS, error) {
	if useEmbedded {
		return fs.Sub(embeddedStatic, "public")
	}
	return os.DirFS("internal/static/public"), nil
}
