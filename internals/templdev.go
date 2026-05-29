package internals

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CleanupStaleTemplDevMode disables templ dev mode when the watch companion file is gone.
// This is only relevant for local `templ generate --watch` runs after an interrupted session.
func CleanupStaleTemplDevMode() {
	if os.Getenv("TEMPL_DEV_MODE") != "true" {
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		return
	}

	sentinelTempl := filepath.Join(wd, "views", "pages", "main", "home", "home.templ")
	companion := companionTxtPathForTemplSource(sentinelTempl)
	if _, err := os.Stat(companion); err != nil {
		DisableTemplDevMode()
	}
}

func DisableTemplDevMode() {
	_ = os.Unsetenv("TEMPL_DEV_MODE")
}

func IsTemplWatchedStringsError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	return strings.Contains(msg, "templ: failed to get watched strings") ||
		strings.Contains(msg, "templ: failed to open /tmp/templ_") ||
		errors.Is(err, os.ErrNotExist)
}

func companionTxtPathForTemplSource(templFileName string) string {
	if strings.HasSuffix(templFileName, "_templ.go") {
		templFileName = strings.TrimSuffix(templFileName, "_templ.go") + ".templ"
	}
	absFileName, err := filepath.Abs(templFileName)
	if err != nil {
		absFileName = templFileName
	}
	absFileName, err = filepath.EvalSymlinks(absFileName)
	if err != nil {
		absFileName = templFileName
	}
	absFileName = normalizeTemplDevPath(absFileName)

	hashedFileName := sha256.Sum256([]byte(absFileName))
	outputFileName := fmt.Sprintf("templ_%s.txt", hex.EncodeToString(hashedFileName[:]))

	root := os.TempDir()
	if os.Getenv("TEMPL_DEV_MODE_ROOT") != "" {
		root = os.Getenv("TEMPL_DEV_MODE_ROOT")
	}

	return filepath.Join(root, outputFileName)
}

func normalizeTemplDevPath(p string) string {
	p = strings.ReplaceAll(filepath.Clean(p), `\`, "/")
	parts := strings.SplitN(p, ":", 2)
	if len(parts) == 2 && len(parts[0]) == 1 {
		drive := strings.ToLower(parts[0])
		p = "/" + drive + parts[1]
	}
	return p
}