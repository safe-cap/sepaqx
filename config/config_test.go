package config

import "testing"

func TestLoad_QRSize_DefaultAndRange(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("QR_SIZE", "")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		if cfg.QRSize != 512 {
			t.Fatalf("QRSize=%d want 512", cfg.QRSize)
		}
	})

	t.Run("valid_custom", func(t *testing.T) {
		t.Setenv("QR_SIZE", "1024")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		if cfg.QRSize != 1024 {
			t.Fatalf("QRSize=%d want 1024", cfg.QRSize)
		}
	})

	t.Run("invalid_low_fallback", func(t *testing.T) {
		t.Setenv("QR_SIZE", "256")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		if cfg.QRSize != 512 {
			t.Fatalf("QRSize=%d want fallback 512", cfg.QRSize)
		}
	})

	t.Run("invalid_high_fallback", func(t *testing.T) {
		t.Setenv("QR_SIZE", "4096")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		if cfg.QRSize != 512 {
			t.Fatalf("QRSize=%d want fallback 512", cfg.QRSize)
		}
	})
}
