package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/safe-cap/sepaqx/config"
)

func ensureTLS(cfg *config.Config, listenAddr string) (string, string, error) {
	certFile := strings.TrimSpace(cfg.TLSCertFile)
	keyFile := strings.TrimSpace(cfg.TLSKeyFile)
	if certFile == "" || keyFile == "" {
		return "", "", fmt.Errorf("TLS_CERT_FILE and TLS_KEY_FILE must be set")
	}

	if fileExists(certFile) && fileExists(keyFile) {
		return certFile, keyFile, nil
	}

	if !cfg.TLSAutoSelfSigned {
		return "", "", fmt.Errorf("tls cert/key not found and TLS_AUTO_SELF_SIGNED is disabled")
	}

	hosts := make([]string, 0, len(cfg.TLSHosts)+4)
	hosts = append(hosts, cfg.TLSHosts...)

	// Add listen host if it is a concrete IP/host (skip 0.0.0.0 / ::)
	if h, _, err := net.SplitHostPort(listenAddr); err == nil && strings.TrimSpace(h) != "" {
		if h != "0.0.0.0" && h != "::" {
			hosts = append(hosts, h)
		}
	}

	// Always include localhost to make local usage painless.
	hosts = append(hosts, "localhost", "127.0.0.1", "::1")
	hosts = uniqStrings(hosts)

	if err := os.MkdirAll(filepath.Dir(certFile), 0o755); err != nil {
		return "", "", fmt.Errorf("create cert dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyFile), 0o755); err != nil {
		return "", "", fmt.Errorf("create key dir: %w", err)
	}

	if err := generateSelfSigned(certFile, keyFile, hosts, cfg.TLSCertDays); err != nil {
		return "", "", err
	}

	return certFile, keyFile, nil
}

func generateSelfSigned(certFile, keyFile string, hosts []string, days int) error {
	if days < 1 {
		days = 365
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return fmt.Errorf("serial: %w", err)
	}

	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   firstNonEmpty(hosts, "localhost"),
			Organization: []string{"sepaqx"},
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(time.Duration(days) * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, h)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("create certificate: %w", err)
	}

	certOut, err := os.OpenFile(certFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("write cert: %w", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		_ = certOut.Close()
		return fmt.Errorf("encode cert: %w", err)
	}
	if err := certOut.Close(); err != nil {
		return fmt.Errorf("close cert: %w", err)
	}

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("write key: %w", err)
	}
	b := x509.MarshalPKCS1PrivateKey(priv)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: b}); err != nil {
		_ = keyOut.Close()
		return fmt.Errorf("encode key: %w", err)
	}
	if err := keyOut.Close(); err != nil {
		return fmt.Errorf("close key: %w", err)
	}

	// Quick sanity check: load what we wrote.
	if _, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
		return fmt.Errorf("invalid generated tls pair: %w", err)
	}

	return nil
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func uniqStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		vv := strings.TrimSpace(v)
		if vv == "" {
			continue
		}
		if _, ok := seen[vv]; ok {
			continue
		}
		seen[vv] = struct{}{}
		out = append(out, vv)
	}
	return out
}

func firstNonEmpty(in []string, def string) string {
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return def
}
