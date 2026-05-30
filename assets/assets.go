package assets

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed css/* fonts/* js/* img/* tinymce/*
var Assets embed.FS

// FS is the filesystem served at /static. Starts as the core embed so the
// binary is self-contained; builds can replace individual files via SetOverlay
// (e.g. a different logo) without editing this package.
var FS fs.FS = Assets

// SetOverlay wraps the current FS so files in overlay win and missing entries
// fall through. Safe to call multiple times to stack layers.
func SetOverlay(overlay fs.FS) {
	FS = overlayFS{overlay: overlay, base: FS}
}

// UseDiskBase replaces the innermost base FS (normally the embed) with one
// rooted on disk, preserving every overlay already stacked on top. Used in dev
// mode (ServeAssetsFromDisk) so CSS/JS/img edits are live without rebuilding,
// without losing overrides registered earlier via SetOverlay.
func UseDiskBase(dir string) {
	FS = replaceBase(FS, os.DirFS(dir))
}

func replaceBase(cur, newBase fs.FS) fs.FS {
	if o, ok := cur.(overlayFS); ok {
		return overlayFS{overlay: o.overlay, base: replaceBase(o.base, newBase)}
	}
	return newBase
}

type overlayFS struct {
	overlay, base fs.FS
}

func (o overlayFS) Open(name string) (fs.File, error) {
	if f, err := o.overlay.Open(name); err == nil {
		return f, nil
	}
	return o.base.Open(name)
}
