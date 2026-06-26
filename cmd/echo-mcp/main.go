package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"echo-mcp/internal/contract"
	"echo-mcp/internal/control"
	"echo-mcp/internal/httpserver"
	"echo-mcp/internal/state"
	"echo-mcp/internal/webhook"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultHTTPAddr = ":8080"
	envHTTPAddr     = "ECHO_MCP_HTTP_ADDR"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	store := state.New()
	controlPlane := control.NewWithWebhookSender(store, loadContractValidator(logger), loadWebhookSender(logger))
	dataPlaneServer := httpserver.New(store, logger)
	controlPlaneServer := control.NewMCPServer(controlPlane)
	httpAddr := httpAddrFromEnvironment(os.Getenv)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errs := make(chan error, 2)
	go func() {
		logger.Info("starting Echo MCP data plane", "addr", httpAddr)
		err := http.ListenAndServe(httpAddr, dataPlaneServer.Handler())
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- err
			return
		}
		errs <- nil
	}()

	go func() {
		logger.Info("starting Echo MCP control plane", "protocol", "mcp", "transport", "stdio")
		errs <- controlPlaneServer.Run(ctx, &mcp.StdioTransport{})
	}()

	if err := <-errs; err != nil {
		logger.Error("Echo MCP stopped", "error", err)
		os.Exit(1)
	}
}

func httpAddrFromEnvironment(getenv func(string) string) string {
	if addr := getenv(envHTTPAddr); addr != "" {
		return addr
	}

	return defaultHTTPAddr
}

func loadContractValidator(logger *slog.Logger) control.ResponseRuleValidator {
	path := os.Getenv("ECHO_MCP_OPENAPI_FILE")
	if path == "" {
		return nil
	}

	validator, err := contract.LoadOpenAPIFile(path)
	if err != nil {
		logger.Error("load OpenAPI contract", "path", path, "error", err)
		os.Exit(1)
	}

	logger.Info("loaded OpenAPI contract", "path", path)
	return validator
}

func loadWebhookSender(logger *slog.Logger) control.WebhookSender {
	endpoints, err := webhook.LoadEndpointFromEnvironment(os.Getenv)
	if err != nil {
		logger.Error("load application webhook endpoint", "error", err)
		os.Exit(1)
	}
	if endpoints == nil {
		return nil
	}

	logger.Info("registered application webhook endpoint")
	return webhook.NewSender(endpoints, nil)
}
