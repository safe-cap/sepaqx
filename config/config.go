package config

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var (
	version = "dev"
	commit  = "unknown"
)

// OverrideBuildInfo allows main to set build metadata.
func OverrideBuildInfo(v, c string) {
	if strings.TrimSpace(v) != "" {
		version = v
	}
	if strings.TrimSpace(c) != "" {
		commit = c
	}
}

type Config struct {
	Version      string
	Commit       string
	ListenIP     string
	ListenPort   int
	KeysFile     string
	LogoMaxRatio float64

	TLSEnabled        bool
	TLSCertFile       string
	TLSKeyFile        string
	TLSHosts          []string
	TLSAutoSelfSigned bool
	TLSCertDays       int

	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	MaxHeaderBytes    int
	MaxBodyBytes      int64
	RateLimitRPS      float64
	RateLimitBurst    int

	AllowQueryAPIKey  bool
	AmountLenientOCR  bool
	TrustedProxyCIDRs []net.IPNet
	RequireKeys       bool
	RequireAPIKey     bool

	AccessLog bool

	CachePNGMaxBytes  int64
	CacheLogoMaxBytes int64
	CacheTTL          time.Duration
	CacheControl      string

	ErrorPNGPath string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	ip := strings.TrimSpace(os.Getenv("LISTEN_IP"))
	if ip == "" {
		ip = "127.0.0.1"
	}

	portStr := strings.TrimSpace(os.Getenv("LISTEN_PORT"))
	if portStr == "" {
		portStr = "8089"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid LISTEN_PORT: %q", portStr)
	}

	keysFile := strings.TrimSpace(os.Getenv("KEYS_FILE"))
	if keysFile == "" {
		keysFile = "./keys.json"
	}

	ratioStr := strings.TrimSpace(os.Getenv("LOGO_MAX_RATIO"))
	ratio := 0.22
	if ratioStr != "" {
		v, err := strconv.ParseFloat(ratioStr, 64)
		if err != nil || v <= 0 || v >= 0.5 {
			return nil, fmt.Errorf("invalid LOGO_MAX_RATIO: %q", ratioStr)
		}
		ratio = v
	}

	certFile := strings.TrimSpace(os.Getenv("TLS_CERT_FILE"))
	if certFile == "" {
		certFile = "./tls/cert.pem"
	}

	tlsEnabled := parseBool(strings.TrimSpace(os.Getenv("TLS_ENABLED")), false)

	keyFile := strings.TrimSpace(os.Getenv("TLS_KEY_FILE"))
	if keyFile == "" {
		keyFile = "./tls/key.pem"
	}

	hostsStr := strings.TrimSpace(os.Getenv("TLS_HOSTS"))
	hosts := splitCSV(hostsStr)
	if len(hosts) == 0 {
		hosts = []string{"localhost", "127.0.0.1"}
	}

	autoStr := strings.TrimSpace(os.Getenv("TLS_AUTO_SELF_SIGNED"))
	auto := true
	if autoStr != "" {
		auto = parseBool(autoStr, true)
	}

	daysStr := strings.TrimSpace(os.Getenv("TLS_CERT_DAYS"))
	days := 365
	if daysStr != "" {
		v, err := strconv.Atoi(daysStr)
		if err != nil || v < 1 || v > 3650 {
			return nil, fmt.Errorf("invalid TLS_CERT_DAYS: %q", daysStr)
		}
		days = v
	}

	rt := mustEnvSeconds("READ_TIMEOUT_SEC", 10)
	wt := mustEnvSeconds("WRITE_TIMEOUT_SEC", 15)
	it := mustEnvSeconds("IDLE_TIMEOUT_SEC", 60)
	rht := mustEnvSeconds("READ_HEADER_TIMEOUT_SEC", 5)

	maxHeaderBytes := mustEnvInt("MAX_HEADER_BYTES", 1<<20, 8<<10, 16<<20)
	maxBodyBytes := int64(mustEnvInt("MAX_BODY_BYTES", 8<<10, 1<<10, 1<<20))
	rateRPS := mustEnvFloat("RATE_LIMIT_RPS", 10, 0, 1000000)
	rateBurst := mustEnvInt("RATE_LIMIT_BURST", 20, 1, 1000000)

	allowQueryAPIKey := parseBool(strings.TrimSpace(os.Getenv("ALLOW_QUERY_API_KEY")), false)
	amountLenientOCR := parseBool(strings.TrimSpace(os.Getenv("AMOUNT_LENIENT_OCR")), false)
	requireKeys := parseBool(strings.TrimSpace(os.Getenv("REQUIRE_KEYS")), false)
	requireAPIKey := parseBool(strings.TrimSpace(os.Getenv("REQUIRE_API_KEY")), false)
	accessLog := parseBool(strings.TrimSpace(os.Getenv("ACCESS_LOG")), false)
	cachePNGBytes := int64(mustEnvInt("CACHE_PNG_MAX_BYTES", 256<<20, 1<<20, 2<<30))
	cacheLogoBytes := int64(mustEnvInt("CACHE_LOGO_MAX_BYTES", 32<<20, 1<<20, 512<<20))
	cacheTTL := mustEnvSeconds("CACHE_TTL_SEC", 900)
	cacheControl := strings.TrimSpace(os.Getenv("CACHE_CONTROL"))
	if cacheControl == "" {
		cacheControl = "private, max-age=60"
	}
	errorPNGPath := strings.TrimSpace(os.Getenv("ERROR_PNG_PATH"))

	trustedCIDRs, err := parseTrustedProxyCIDRs(strings.TrimSpace(os.Getenv("TRUSTED_PROXY_CIDRS")))
	if err != nil {
		return nil, err
	}

	return &Config{
		Version:      version,
		Commit:       commit,
		ListenIP:     ip,
		ListenPort:   port,
		KeysFile:     keysFile,
		LogoMaxRatio: ratio,

		TLSEnabled:        tlsEnabled,
		TLSCertFile:       certFile,
		TLSKeyFile:        keyFile,
		TLSHosts:          hosts,
		TLSAutoSelfSigned: auto,
		TLSCertDays:       days,

		ReadTimeout:       rt,
		WriteTimeout:      wt,
		IdleTimeout:       it,
		ReadHeaderTimeout: rht,
		MaxHeaderBytes:    maxHeaderBytes,
		MaxBodyBytes:      maxBodyBytes,
		RateLimitRPS:      rateRPS,
		RateLimitBurst:    rateBurst,
		AllowQueryAPIKey:  allowQueryAPIKey,
		AmountLenientOCR:  amountLenientOCR,
		TrustedProxyCIDRs: trustedCIDRs,
		RequireKeys:       requireKeys,
		RequireAPIKey:     requireAPIKey,
		AccessLog:         accessLog,
		CachePNGMaxBytes:  cachePNGBytes,
		CacheLogoMaxBytes: cacheLogoBytes,
		CacheTTL:          cacheTTL,
		CacheControl:      cacheControl,

		ErrorPNGPath: errorPNGPath,
	}, nil
}

func mustEnvSeconds(name string, def int) time.Duration {
	s := strings.TrimSpace(os.Getenv(name))
	if s == "" {
		return time.Duration(def) * time.Second
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 || v > 600 {
		log.Printf("WARN: %s=%q is invalid (must be integer 1..600); using default %ds", name, s, def)
		return time.Duration(def) * time.Second
	}
	return time.Duration(v) * time.Second
}

func mustEnvInt(name string, def, min, max int) int {
	s := strings.TrimSpace(os.Getenv(name))
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < min || v > max {
		log.Printf("WARN: %s=%q is invalid (must be integer %d..%d); using default %d", name, s, min, max, def)
		return def
	}
	return v
}

func mustEnvFloat(name string, def, min, max float64) float64 {
	s := strings.TrimSpace(os.Getenv(name))
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < min || v > max {
		log.Printf("WARN: %s=%q is invalid (must be float %.6g..%.6g); using default %.6g", name, s, min, max, def)
		return def
	}
	return v
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func parseBool(s string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

func parseTrustedProxyCIDRs(s string) ([]net.IPNet, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]net.IPNet, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		_, cidr, err := net.ParseCIDR(v)
		if err != nil {
			return nil, fmt.Errorf("invalid TRUSTED_PROXY_CIDRS entry: %q", v)
		}
		out = append(out, *cidr)
	}
	return out, nil
}
