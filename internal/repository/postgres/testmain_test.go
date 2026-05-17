package postgres

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain переносит cwd в корень модуля: goose.UpContext использует путь "migrations" относительно cwd,
// а go test запускает бинарник из каталога пакета.
func TestMain(m *testing.M) {
	chdirModuleRoot()
	os.Exit(m.Run())
}

// chdirModuleRoot переходит в корень модуля по цепочке родительских каталогов.
func chdirModuleRoot() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			if err := os.Chdir(wd); err != nil {
				panic(err)
			}
			return
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return
		}
		wd = parent
	}
}
