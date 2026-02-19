package keys

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile_QRSizeValidation(t *testing.T) {
	dir := t.TempDir()
	logo := filepath.Join(dir, "logo.png")
	if err := os.WriteFile(logo, []byte{1, 2, 3}, 0o644); err != nil {
		t.Fatalf("write logo: %v", err)
	}

	keysPath := filepath.Join(dir, "keys.json")
	content := `{
  "keys": [
    { "key": "k1", "name": "n1", "qr_size": 1024, "logo_path": "` + logo + `" },
    { "key": "k2", "name": "n2", "qr_size": 4096, "logo_path": "` + logo + `" }
  ]
}`
	if err := os.WriteFile(keysPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write keys: %v", err)
	}

	store, err := LoadFromFile(keysPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}

	k1, ok := store.Get("k1")
	if !ok {
		t.Fatalf("missing key k1")
	}
	if k1.QRSize != 1024 {
		t.Fatalf("k1 QRSize=%d want 1024", k1.QRSize)
	}

	k2, ok := store.Get("k2")
	if !ok {
		t.Fatalf("missing key k2")
	}
	if k2.QRSize != 0 {
		t.Fatalf("k2 QRSize=%d want 0 (disabled override)", k2.QRSize)
	}
}
