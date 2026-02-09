package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/safe-cap/sepaqx/config"
	"github.com/safe-cap/sepaqx/keys"
	"github.com/safe-cap/sepaqx/server"
)

var version = "dev"
var commit = "none"
var date = "unknown"

//go:embed img/error.png
var defaultErrorPNG []byte

func main() {
	showVersion := flag.Bool("v", false, "print version and exit")
	showVersionLong := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion || *showVersionLong {
		fmt.Printf("SepaQX version %s\n", version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	config.OverrideBuildInfo(version, commit)

	log.Printf("SepaQX version %s", version)

	if isPprofEnabled() {
		pprofAddr := strings.TrimSpace(os.Getenv("PPROF_LISTEN"))
		if pprofAddr == "" {
			pprofAddr = "127.0.0.1:6060"
		}
		go func() {
			log.Printf("pprof listening on http://%s/debug/pprof/", pprofAddr)
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				log.Printf("pprof server stopped: %v", err)
			}
		}()
	}

	keyStore, err := keys.LoadFromFile(cfg.KeysFile)
	if err != nil {
		if cfg.RequireKeys {
			log.Fatalf("keys load failed (REQUIRE_KEYS=1): %v", err)
		}
		log.Printf("keys load failed, auth disabled: %v", err)
		keyStore = keys.NewEmpty()
	}
	if keyStore.Len() == 0 {
		if cfg.RequireKeys {
			log.Fatalf("no valid keys loaded (REQUIRE_KEYS=1)")
		}
		log.Printf("no valid keys loaded; auth mode disabled")
		if cfg.RequireAPIKey {
			log.Printf("REQUIRE_API_KEY enabled but no valid keys loaded; all requests will be unauthorized")
		}
	}

	srv := server.New(cfg, keyStore, defaultErrorPNG)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stop
		log.Printf("shutdown requested")
		_ = srv.Shutdown()
	}()

	if err := srv.Run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func isPprofEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PPROF_ENABLED")))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
