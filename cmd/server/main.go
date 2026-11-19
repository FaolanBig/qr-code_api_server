package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/FaolanBig/qr-code_api_server/internal/config"
	"github.com/FaolanBig/qr-code_api_server/internal/httpserver"
	qr "github.com/FaolanBig/qr-code_api_server/internal/qrcode"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	fallbackLogo, err := os.ReadFile(cfg.FallbackLogoPath)
	if err != nil {
		logger.Error("fallback logo could not be read", "path", cfg.FallbackLogoPath, "error", err)
		os.Exit(1)
	}
	if _, err := qr.ValidatePNGLogo(fallbackLogo, cfg.LogoMaxBytes); err != nil {
		logger.Error("fallback logo is invalid", "path", cfg.FallbackLogoPath, "error", err)
		os.Exit(1)
	}

	openAPISpec, err := os.ReadFile(cfg.OpenAPIPath)
	if err != nil {
		logger.Error("OpenAPI specification could not be read", "path", cfg.OpenAPIPath, "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr: cfg.Address,
		Handler: httpserver.New(
			cfg.APIKey,
			logger,
			httpserver.WithFallbackLogo(fallbackLogo),
			httpserver.WithLogoMaxBytes(cfg.LogoMaxBytes),
			httpserver.WithOpenAPISpec(openAPISpec),
		).Handler(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("QR code API listening", "address", cfg.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownSignal
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
	logger.Info("server stopped")
}
