package keys

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

type Palette struct {
	FG string `json:"fg"`
	BG string `json:"bg"`
}

type Gradient struct {
	From  string  `json:"from"`
	To    string  `json:"to"`
	Angle float64 `json:"angle"`
}

type KeyConfig struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	QRSize       int      `json:"qr_size"`
	LogoPath     string   `json:"logo_path"`
	LogoBGShape  string   `json:"logo_bg_shape"`
	Palette      Palette  `json:"palette"`
	FGGradient   Gradient `json:"fg_gradient"`
	BGGradient   Gradient `json:"bg_gradient"`
	CornerRadius int      `json:"corner_radius"`
	ModuleStyle  string   `json:"module_style"`
	ModuleRadius float64  `json:"module_radius"`
	QuietZone    int      `json:"quiet_zone"`
}

type storeFile struct {
	Keys []KeyConfig `json:"keys"`
}

type Store struct {
	byKey map[string]KeyConfig
}

var reHexColor = regexp.MustCompile(`^#?[0-9a-fA-F]{6}$`)

func NewEmpty() *Store {
	return &Store{byKey: make(map[string]KeyConfig)}
}

func LoadFromFile(path string) (*Store, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read keys file: %w", err)
	}

	var sf storeFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return nil, fmt.Errorf("parse keys file: %w", err)
	}

	byKey := make(map[string]KeyConfig)
	for _, k := range sf.Keys {
		kk := strings.TrimSpace(k.Key)
		if kk == "" {
			log.Printf("keys: skipping entry with empty key (name=%q)", k.Name)
			continue
		}
		k.LogoPath = strings.TrimSpace(k.LogoPath)
		if k.LogoPath != "" && !isReadableFile(k.LogoPath) {
			log.Printf("keys: logo not readable, disabling (name=%q, logo=%q)", k.Name, k.LogoPath)
			k.LogoPath = ""
		}

		fg := normalizeHex(k.Palette.FG)
		if k.Palette.FG != "" && fg == "" {
			log.Printf("keys: invalid palette fg, disabling (name=%q, fg=%q)", k.Name, k.Palette.FG)
		}
		bg := normalizeHex(k.Palette.BG)
		if k.Palette.BG != "" && bg == "" {
			log.Printf("keys: invalid palette bg, disabling (name=%q, bg=%q)", k.Name, k.Palette.BG)
		}
		k.Palette.FG = fg
		k.Palette.BG = bg

		k.FGGradient.From, k.FGGradient.To = normalizeGradient(k.Name, "fg_gradient", k.FGGradient.From, k.FGGradient.To)
		k.BGGradient.From, k.BGGradient.To = normalizeGradient(k.Name, "bg_gradient", k.BGGradient.From, k.BGGradient.To)

		k.LogoBGShape = normalizeLogoBGShape(k.LogoBGShape)
		k.ModuleStyle = normalizeModuleStyle(k.ModuleStyle)

		if k.ModuleRadius < 0 || k.ModuleRadius > 0.5 {
			log.Printf("keys: invalid module_radius, disabling (name=%q, module_radius=%v)", k.Name, k.ModuleRadius)
			k.ModuleRadius = 0
		}
		if k.CornerRadius < 0 {
			log.Printf("keys: invalid corner_radius, disabling (name=%q, corner_radius=%v)", k.Name, k.CornerRadius)
			k.CornerRadius = 0
		}
		if k.QuietZone < 0 || k.QuietZone > 20 {
			log.Printf("keys: invalid quiet_zone, disabling (name=%q, quiet_zone=%v)", k.Name, k.QuietZone)
			k.QuietZone = 0
		}
		if k.QRSize != 0 && (k.QRSize < 512 || k.QRSize > 2048) {
			log.Printf("keys: invalid qr_size, disabling per-key override (name=%q, qr_size=%v)", k.Name, k.QRSize)
			k.QRSize = 0
		}

		byKey[kk] = k
	}

	if len(byKey) == 0 {
		return nil, fmt.Errorf("no valid keys in file")
	}

	return &Store{byKey: byKey}, nil
}

func normalizeGradient(name, field, from, to string) (string, string) {
	f := normalizeHex(from)
	t := normalizeHex(to)
	if (from != "" && f == "") || (to != "" && t == "") {
		log.Printf("keys: invalid %s, disabling (name=%q, from=%q, to=%q)", field, name, from, to)
		return "", ""
	}
	return f, t
}

func normalizeModuleStyle(s string) string {
	v := strings.TrimSpace(strings.ToLower(s))
	switch v {
	case "", "square":
		return "square"
	case "rounded", "blob":
		return v
	default:
		return "square"
	}
}

func normalizeLogoBGShape(s string) string {
	v := strings.TrimSpace(strings.ToLower(s))
	switch v {
	case "", "square":
		return "square"
	case "circle":
		return "circle"
	default:
		return "square"
	}
}

func (s *Store) Get(apiKey string) (KeyConfig, bool) {
	v, ok := s.byKey[apiKey]
	return v, ok
}

func (s *Store) Len() int {
	if s == nil {
		return 0
	}
	return len(s.byKey)
}

func normalizeHex(s string) string {
	v := strings.TrimSpace(s)
	if v == "" {
		return ""
	}
	if !reHexColor.MatchString(v) {
		return ""
	}
	if v[0] != '#' {
		return "#" + strings.ToLower(v)
	}
	return strings.ToLower(v)
}

func isReadableFile(path string) bool {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
