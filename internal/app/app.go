package app

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/config"
	"github.com/nie-crossroads/clipbridge/internal/parser"
	"github.com/nie-crossroads/clipbridge/internal/parser/bilibili"
	"github.com/nie-crossroads/clipbridge/internal/parser/share"
	"github.com/nie-crossroads/clipbridge/internal/repository/mysql"
	"github.com/nie-crossroads/clipbridge/internal/service"
	httptransport "github.com/nie-crossroads/clipbridge/internal/transport/http"
	"github.com/nie-crossroads/clipbridge/web"
)

func Serve(ctx context.Context, cfg config.Config, store *mysql.Store) error {
	_, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return err
	}
	if !cfg.SecureCookie && !loopbackAddress(cfg.Addr) {
		return errors.New("CLIPBRIDGE_SECURE_COOKIE=false requires a loopback listen address")
	}
	if cfg.LocalAdminUsername != "" || cfg.LocalAdminPassword != "" {
		if err := syncLocalAdmin(ctx, cfg, store); err != nil {
			return err
		}
	}
	sessions, err := service.NewSessions(store)
	if err != nil {
		return err
	}
	parse := service.NewParseService(parser.NewRegistry(append([]parser.Parser{bilibili.New()}, share.Adapters()...)...), store)
	parse.LoadSettings(ctx)
	go parse.MaintainLogs(ctx)
	server := &http.Server{
		Addr: cfg.Addr, Handler: httptransport.New(parse, sessions, web.Assets(), cfg.SecureCookie),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 35 * time.Second,
		IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10,
	}
	done := make(chan error, 1)
	go func() {
		slog.Info("ClipBridge listening", "address", cfg.Addr)
		done <- server.ListenAndServe()
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}

func ValidUsername(value string) bool {
	return strings.TrimSpace(value) == value && len([]rune(value)) >= 1 && len([]rune(value)) <= 64
}
