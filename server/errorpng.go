package server

import (
	"log"
	"os"
)

func loadErrorPNG(path string, fallback []byte) []byte {
	if path != "" {
		b, err := os.ReadFile(path)
		if err == nil && len(b) > 0 {
			return b
		}
		log.Printf("error png load failed, using default: %v", err)
	}
	if len(fallback) == 0 {
		return nil
	}
	return fallback
}
