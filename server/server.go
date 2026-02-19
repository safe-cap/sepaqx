package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/safe-cap/sepaqx/auth"
	"github.com/safe-cap/sepaqx/config"
	"github.com/safe-cap/sepaqx/keys"
	"github.com/safe-cap/sepaqx/qr"
	"github.com/safe-cap/sepaqx/validate"
)

type Server struct {
	cfg        *config.Config
	keys       *keys.Store
	ready      bool
	readyMsg   string
	httpSrv    *http.Server
	limiter    *ipLimiter
	pngCache   *pngCache
	logoCache  *logoCache
	logLimiter *logLimiter
	errorPNG   []byte
}

func New(cfg *config.Config, keyStore *keys.Store, defaultErrorPNG []byte) *Server {
	s := &Server{
		cfg:      cfg,
		keys:     keyStore,
		ready:    true,
		readyMsg: "ready",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReady)
	mux.HandleFunc("/version", s.handleVersion)
	mux.HandleFunc("/sepa-qr", s.handleSEPA)
	mux.HandleFunc("/sepa-qr/validate", s.handleValidate)

	handler := http.Handler(mux)
	if cfg.AccessLog {
		handler = s.accessLogMiddleware(handler)
	}
	handler = s.requestIDMiddleware(handler)

	addr := net.JoinHostPort(cfg.ListenIP, fmt.Sprintf("%d", cfg.ListenPort))

	s.httpSrv = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
	s.withTrustedProxyCtx()
	// Disable HTTP/2 so responses use HTTP/1.1 even over TLS.
	s.httpSrv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))

	s.limiter = newIPLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst, 5*time.Minute)
	s.pngCache = newPNGCache(cfg.CachePNGMaxBytes, cfg.CacheTTL)
	s.logoCache = newLogoCache(cfg.CacheLogoMaxBytes)
	s.logLimiter = newLogLimiter(10 * time.Second)
	s.errorPNG = loadErrorPNG(cfg.ErrorPNGPath, defaultErrorPNG)

	return s
}

func (s *Server) Run() error {
	if !s.cfg.TLSEnabled {
		log.Printf("sepaqx listening on http://%s/sepa-qr", s.httpSrv.Addr)
		err := s.httpSrv.ListenAndServe()
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}

	certFile, keyFile, err := ensureTLS(s.cfg, s.httpSrv.Addr)
	if err != nil {
		return err
	}
	s.httpSrv.TLSConfig = &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
	}
	log.Printf("sepaqx listening on https://%s/sepa-qr", s.httpSrv.Addr)
	err = s.httpSrv.ListenAndServeTLS(certFile, keyFile)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if s.ready {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ready\n"))
		return
	}
	if wantsJSONResponse(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":         false,
			"status":     "not_ready",
			"reason":     s.readyMsg,
			"request_id": requestIDFromContext(r.Context()),
		})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte("not ready: " + s.readyMsg + "\n"))
}

func (s *Server) SetReadiness(ready bool, reason string) {
	s.ready = ready
	if strings.TrimSpace(reason) == "" {
		if ready {
			s.readyMsg = "ready"
		} else {
			s.readyMsg = "not ready"
		}
		return
	}
	s.readyMsg = strings.TrimSpace(reason)
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"version":             s.cfg.Version,
		"commit":              s.cfg.Commit,
		"tls_enabled":         s.cfg.TLSEnabled,
		"allow_query_api_key": s.cfg.AllowQueryAPIKey,
		"require_api_key":     s.cfg.RequireAPIKey,
	})
}

func (s *Server) handleSEPA(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		allow := []string{http.MethodPost}
		allow = append(allow, http.MethodGet, http.MethodHead)
		allow = append(allow, http.MethodOptions)
		w.Header().Set("Allow", strings.Join(allow, ", "))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost &&
		!(r.Method == http.MethodGet || r.Method == http.MethodHead) {
		s.writeError(w, r, CodeMethodNotAllowed, "", "")
		return
	}

	if !s.limiter.Allow(clientIP(r)) {
		s.writeError(w, r, CodeRateLimited, "", "")
		return
	}

	apiKey := auth.ExtractAPIKey(r, s.cfg.AllowQueryAPIKey)

	// Public mode: if no key is provided at all, generate a standard QR.
	isPublic := strings.TrimSpace(apiKey) == ""
	if s.cfg.RequireAPIKey && isPublic {
		s.writeError(w, r, CodeUnauthorized, "unauthorized", "")
		return
	}

	// Allow a bare HEAD request to succeed without validating parameters,
	// but only when public access is allowed.
	if r.Method == http.MethodHead && r.URL.RawQuery == "" {
		s.writePNGHeadersOnly(w)
		return
	}

	var (
		keyCfg keys.KeyConfig
		ok     bool
	)

	if !isPublic {
		keyCfg, ok = s.keys.Get(apiKey)
		if !ok {
			// Key was provided but invalid -> explicit 401 (do not fall back to public).
			s.writeError(w, r, CodeUnauthorized, "unauthorized", "")
			return
		}
	}

	var in validate.Input
	switch r.Method {
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			s.logLimiter.Logf(string(CodeInvalidJSON), "invalid json body: %v", err)
			s.writeError(w, r, CodeInvalidJSON, "invalid json body", "")
			return
		}
		var extra any
		if err := dec.Decode(&extra); err != io.EOF {
			s.logLimiter.Logf(string(CodeInvalidJSON), "invalid json body: trailing data")
			s.writeError(w, r, CodeInvalidJSON, "invalid json body", "")
			return
		}
	case http.MethodGet, http.MethodHead:
		q := r.URL.Query()
		parsedIn, parseErr := inputFromQuery(q)
		if parseErr != nil {
			s.logLimiter.Logf(string(CodeInvalidInput), "invalid input: %v", parseErr)
			field := fieldFromValidationError(parseErr.Error())
			s.writeError(w, r, CodeInvalidInput, parseErr.Error(), field)
			return
		}
		in = parsedIn
	}

	cleaned, err := validate.CleanAndValidate(in)
	if err != nil {
		s.logLimiter.Logf(string(CodeInvalidInput), "invalid input: %v", err)
		field := fieldFromValidationError(err.Error())
		s.writeError(w, r, CodeInvalidInput, err.Error(), field)
		return
	}

	payload, err := qr.BuildEPCPayload(
		cleaned.Name,
		cleaned.IBAN,
		cleaned.BIC,
		cleaned.AmountCents,
		cleaned.Purpose,
		cleaned.RemittanceReference,
		cleaned.RemittanceText,
		cleaned.Information,
	)
	if err != nil {
		s.writeError(w, r, CodePayloadBuildFailed, "payload build failed", "")
		return
	}

	// QR generation options:
	// - Public: size from global QR_SIZE, ECC=M, margin=4 (library default), no logo, black on transparent.
	// - Auth:  size from global QR_SIZE (or per-key qr_size override), ECC=M unless logo is used (then ECC=H), palette/logo only via key.
	withLogo := !isPublic && keyCfg.LogoPath != ""
	opt := qr.DefaultPublicOptions()
	opt.Size = s.cfg.QRSize
	if !isPublic {
		opt = qr.DefaultAuthOptions(withLogo)
		opt.Size = s.cfg.QRSize
		if keyCfg.QRSize > 0 {
			opt.Size = keyCfg.QRSize
		}
	}

	cacheKey := buildCacheKey(isPublic, cleaned, keyCfg, s.cfg.LogoMaxRatio, opt)
	if cached, ok := s.pngCache.Get(cacheKey); ok {
		s.writePNG(w, r, cached)
		return
	}

	var pngBytes []byte
	if !isPublic && (keyCfg.ModuleStyle == "rounded" || keyCfg.ModuleStyle == "blob" || keyCfg.CornerRadius > 0 || keyCfg.QuietZone > 0) {
		style := qr.Style{
			CornerRadius: keyCfg.CornerRadius,
			ModuleStyle:  keyCfg.ModuleStyle,
			ModuleRadius: keyCfg.ModuleRadius,
			QuietZone:    keyCfg.QuietZone,
		}
		pngBytes, err = qr.MakeQRStyled(payload, opt, style)
	} else {
		pngBytes, err = qr.MakeQR(payload, opt)
	}
	if err != nil {
		s.writeError(w, r, CodeQREncodeFailed, "qr encode failed", "")
		return
	}

	if isPublic {
		// Standard public output: black on transparent background, no logo.
		recolored, err := qr.Recolor(pngBytes, "#000000", "transparent")
		if err == nil {
			pngBytes = recolored
		}
	} else {
		// Apply palette/gradient (auth only)
		if keyCfg.Palette.FG != "" || keyCfg.Palette.BG != "" || keyCfg.FGGradient.From != "" || keyCfg.BGGradient.From != "" {
			fg := keyCfg.Palette.FG
			bg := keyCfg.Palette.BG
			if fg == "" {
				fg = "#000000"
			}
			if bg == "" {
				bg = "#ffffff"
			}
			var fgGrad *qr.GradientSpec
			var bgGrad *qr.GradientSpec
			if keyCfg.FGGradient.From != "" && keyCfg.FGGradient.To != "" {
				fgGrad = &qr.GradientSpec{From: keyCfg.FGGradient.From, To: keyCfg.FGGradient.To, Angle: keyCfg.FGGradient.Angle}
			}
			if keyCfg.BGGradient.From != "" && keyCfg.BGGradient.To != "" {
				bgGrad = &qr.GradientSpec{From: keyCfg.BGGradient.From, To: keyCfg.BGGradient.To, Angle: keyCfg.BGGradient.Angle}
			}
			recolored, err := qr.RecolorGradient(pngBytes, fg, bg, fgGrad, bgGrad)
			if err == nil {
				pngBytes = recolored
			} else {
				s.logLimiter.Logf("recolor:"+keyCfg.Name, "recolor failed for key=%s: %v", keyCfg.Name, err)
			}
		}

		// Overlay logo (auth only). ECC was increased above if logo is used.
		if keyCfg.LogoPath != "" {
			logoImg, ok := s.logoCache.Get(keyCfg.LogoPath)
			if !ok {
				loaded, err := loadLogoImage(keyCfg.LogoPath)
				if err == nil {
					s.logoCache.Set(keyCfg.LogoPath, loaded)
					logoImg = loaded
					ok = true
				} else {
					s.logLimiter.Logf("logo-load:"+keyCfg.Name, "overlay logo failed for key=%s: %v", keyCfg.Name, err)
				}
			}
			if ok {
				withLogo, err := qr.OverlayLogoImage(pngBytes, logoImg, s.cfg.LogoMaxRatio, keyCfg.LogoBGShape)
				if err == nil {
					pngBytes = withLogo
				} else {
					s.logLimiter.Logf("logo-overlay:"+keyCfg.Name, "overlay logo failed for key=%s: %v", keyCfg.Name, err)
				}
			}
		}
	}

	s.pngCache.Set(cacheKey, pngBytes)
	s.writePNG(w, r, pngBytes)
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, CodeMethodNotAllowed, "", "", requestIDFromContext(r.Context()))
		return
	}
	if !s.limiter.Allow(clientIP(r)) {
		s.writeJSONError(w, CodeRateLimited, "", "", requestIDFromContext(r.Context()))
		return
	}
	apiKey := auth.ExtractAPIKey(r, s.cfg.AllowQueryAPIKey)
	isPublic := strings.TrimSpace(apiKey) == ""
	if s.cfg.RequireAPIKey && isPublic {
		s.writeJSONError(w, CodeUnauthorized, "unauthorized", "", requestIDFromContext(r.Context()))
		return
	}
	if !isPublic {
		if _, ok := s.keys.Get(apiKey); !ok {
			s.writeJSONError(w, CodeUnauthorized, "unauthorized", "", requestIDFromContext(r.Context()))
			return
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var in validate.Input
	if err := dec.Decode(&in); err != nil {
		s.logLimiter.Logf(string(CodeInvalidJSON), "validate: invalid json body: %v", err)
		s.writeJSONValidation(w, false, CodeInvalidJSON, "invalid json body", "", requestIDFromContext(r.Context()))
		return
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		s.logLimiter.Logf(string(CodeInvalidJSON), "validate: invalid json body: trailing data")
		s.writeJSONValidation(w, false, CodeInvalidJSON, "invalid json body", "", requestIDFromContext(r.Context()))
		return
	}
	_, err := validate.CleanAndValidate(in)
	if err != nil {
		s.logLimiter.Logf(string(CodeInvalidInput), "validate: invalid input: %v", err)
		field := fieldFromValidationError(err.Error())
		s.writeJSONValidation(w, false, CodeInvalidInput, err.Error(), field, requestIDFromContext(r.Context()))
		return
	}
	s.writeJSONValidation(w, true, "", "", "", requestIDFromContext(r.Context()))
}

func (s *Server) writeJSONValidation(w http.ResponseWriter, ok bool, code ErrorCode, details, field, reqID string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if ok {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":         true,
			"request_id": reqID,
		})
		return
	}
	if code == "" {
		code = CodeInvalidInput
	}
	w.WriteHeader(errorStatus(code))
	if details == "" {
		details = string(code)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":         false,
		"error_code": string(code),
		"details":    details,
		"field":      field,
		"request_id": reqID,
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || strings.TrimSpace(host) == "" {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return host
	}

	// Trust X-Forwarded-For / X-Real-IP only if the immediate peer is trusted.
	if isTrustedProxy(ip, r.Context()) {
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			parts := strings.Split(xff, ",")
			for _, p := range parts {
				cand := net.ParseIP(strings.TrimSpace(p))
				if cand != nil {
					return cand.String()
				}
			}
		}
		if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
			if cand := net.ParseIP(xr); cand != nil {
				return cand.String()
			}
		}
	}
	return ip.String()
}

type trustedProxyKey struct{}

func (s *Server) withTrustedProxyCtx() {
	if len(s.cfg.TrustedProxyCIDRs) == 0 {
		return
	}
	baseHandler := s.httpSrv.Handler
	s.httpSrv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), trustedProxyKey{}, s.cfg.TrustedProxyCIDRs)
		baseHandler.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isTrustedProxy(ip net.IP, ctx context.Context) bool {
	v := ctx.Value(trustedProxyKey{})
	if v == nil {
		return false
	}
	cidrs, ok := v.([]net.IPNet)
	if !ok {
		return false
	}
	for _, c := range cidrs {
		if c.Contains(ip) {
			return true
		}
	}
	return false
}

type accessLogWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *accessLogWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *accessLogWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func (s *Server) accessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &accessLogWriter{ResponseWriter: w}
		next.ServeHTTP(lw, r)
		latency := time.Since(start)
		status := lw.status
		if status == 0 {
			status = http.StatusOK
		}
		log.Printf(
			"access ip=%s method=%s path=%s status=%d bytes=%d latency=%s",
			clientIP(r), r.Method, r.URL.Path, status, lw.bytes, latency,
		)
	})
}

func buildCacheKey(isPublic bool, cleaned *validate.Clean, keyCfg keys.KeyConfig, ratio float64, opt qr.Options) string {
	var b strings.Builder
	if isPublic {
		b.WriteString("p|")
	} else {
		b.WriteString("a|")
		b.WriteString(keyCfg.Name)
		b.WriteString("|")
		b.WriteString(keyCfg.Palette.FG)
		b.WriteString("|")
		b.WriteString(keyCfg.Palette.BG)
		b.WriteString("|")
		b.WriteString(keyCfg.FGGradient.From)
		b.WriteString("|")
		b.WriteString(keyCfg.FGGradient.To)
		b.WriteString("|")
		b.WriteString(fmt.Sprintf("%.3f", keyCfg.FGGradient.Angle))
		b.WriteString("|")
		b.WriteString(keyCfg.BGGradient.From)
		b.WriteString("|")
		b.WriteString(keyCfg.BGGradient.To)
		b.WriteString("|")
		b.WriteString(fmt.Sprintf("%.3f", keyCfg.BGGradient.Angle))
		b.WriteString("|")
		b.WriteString(keyCfg.LogoPath)
		b.WriteString("|")
		b.WriteString(keyCfg.LogoBGShape)
		b.WriteString("|")
		b.WriteString(keyCfg.ModuleStyle)
		b.WriteString("|")
		b.WriteString(fmt.Sprintf("%.3f", keyCfg.ModuleRadius))
		b.WriteString("|")
		b.WriteString(fmt.Sprintf("%d", keyCfg.CornerRadius))
		b.WriteString("|")
		b.WriteString(fmt.Sprintf("%d", keyCfg.QuietZone))
		b.WriteString("|")
	}
	b.WriteString(cleaned.Name)
	b.WriteString("|")
	b.WriteString(cleaned.IBAN)
	b.WriteString("|")
	b.WriteString(cleaned.BIC)
	b.WriteString("|")
	b.WriteString(cleaned.Purpose)
	b.WriteString("|")
	b.WriteString(cleaned.RemittanceReference)
	b.WriteString("|")
	b.WriteString(cleaned.RemittanceText)
	b.WriteString("|")
	b.WriteString(cleaned.Information)
	b.WriteString("|")
	b.WriteString(fmt.Sprintf("%d", cleaned.AmountCents))
	b.WriteString("|")
	b.WriteString(fmt.Sprintf("%d", opt.Size))
	b.WriteString("|")
	b.WriteString(fmt.Sprintf("%d", opt.ECC))
	b.WriteString("|")
	b.WriteString(fmt.Sprintf("%.4f", ratio))

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func loadLogoImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func etagForBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Server) writePNG(w http.ResponseWriter, r *http.Request, pngBytes []byte) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", s.cfg.CacheControl)
	w.Header().Set("ETag", `"`+etagForBytes(pngBytes)+`"`)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pngBytes)))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(pngBytes)
}

func (s *Server) writeErrorPNG(w http.ResponseWriter, r *http.Request, status int, code string) {
	if status < 400 {
		status = http.StatusBadRequest
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if code != "" {
		w.Header().Set("X-Error-Code", code)
	}
	if reqID := requestIDFromContext(r.Context()); reqID != "" {
		w.Header().Set("X-Request-ID", reqID)
	}
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return
	}
	if len(s.errorPNG) == 0 {
		return
	}
	_, _ = w.Write(s.errorPNG)
}

type requestIDKey struct{}

func (s *Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := newRequestID()
		ctx := context.WithValue(r.Context(), requestIDKey{}, reqID)
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestIDFromContext(ctx context.Context) string {
	v := ctx.Value(requestIDKey{})
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func fieldFromValidationError(msg string) string {
	if strings.HasPrefix(msg, "unsupported currency: ") {
		return "amount"
	}
	if msg == "unsupported amount_format" {
		return "amount_format"
	}
	if strings.HasPrefix(msg, "duplicate query parameter: ") {
		return strings.TrimSpace(strings.TrimPrefix(msg, "duplicate query parameter: "))
	}
	switch msg {
	case "unsupported scheme":
		return "scheme"
	case "name is required":
		return "name"
	case "iban is required", "invalid iban":
		return "iban"
	case "bic is required", "invalid bic":
		return "bic"
	case "amount is required", "invalid amount", "amount must be > 0", "amount too large":
		return "amount"
	case "remittance_reference and remittance_text are mutually exclusive":
		return "remittance_reference"
	default:
		return ""
	}
}

func inputFromQuery(q url.Values) (validate.Input, error) {
	var in validate.Input
	var err error

	if in.Scheme, err = singleQueryParam(q, "scheme"); err != nil {
		return validate.Input{}, err
	}
	if in.Name, err = singleQueryParam(q, "name"); err != nil {
		return validate.Input{}, err
	}
	if in.IBAN, err = singleQueryParam(q, "iban"); err != nil {
		return validate.Input{}, err
	}
	if in.BIC, err = singleQueryParam(q, "bic"); err != nil {
		return validate.Input{}, err
	}
	if in.Amount, err = singleQueryParam(q, "amount"); err != nil {
		return validate.Input{}, err
	}
	if in.AmountFormat, err = singleQueryParam(q, "amount_format"); err != nil {
		return validate.Input{}, err
	}
	if in.Purpose, err = singleQueryParam(q, "purpose"); err != nil {
		return validate.Input{}, err
	}
	if in.RemittanceReference, err = singleQueryParam(q, "remittance_reference"); err != nil {
		return validate.Input{}, err
	}
	if in.RemittanceText, err = singleQueryParam(q, "remittance_text"); err != nil {
		return validate.Input{}, err
	}
	if in.Information, err = singleQueryParam(q, "information"); err != nil {
		return validate.Input{}, err
	}

	return in, nil
}

func singleQueryParam(q url.Values, key string) (string, error) {
	values, ok := q[key]
	if !ok || len(values) == 0 {
		return "", nil
	}
	if len(values) > 1 {
		return "", fmt.Errorf("duplicate query parameter: %s", key)
	}
	return values[0], nil
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, code ErrorCode, details, field string) {
	if wantsJSONError(r) {
		s.writeJSONError(w, code, details, field, requestIDFromContext(r.Context()))
		return
	}
	s.writeErrorPNG(w, r, errorStatus(code), string(code))
}

func wantsJSONError(r *http.Request) bool {
	if strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format"))) == "json" {
		return true
	}
	return wantsJSONResponse(r)
}

func wantsJSONResponse(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "application/json") && !strings.Contains(accept, "image/png")
}

func (s *Server) writeJSONError(w http.ResponseWriter, code ErrorCode, details, field, reqID string) {
	if code == "" {
		code = CodeInvalidInput
	}
	if details == "" {
		details = string(code)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(errorStatus(code))
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":         false,
		"error_code": string(code),
		"details":    details,
		"field":      field,
		"request_id": reqID,
	})
}

func (s *Server) writePNGHeadersOnly(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", s.cfg.CacheControl)
	w.Header().Set("Content-Length", "0")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
