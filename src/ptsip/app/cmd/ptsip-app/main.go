package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/Kinirin/PTSIP/src/ptsip/app"
)

var toolVersion = "development"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	flags := flag.NewFlagSet("ptsip-app", flag.ContinueOnError)
	host := flags.String("host", env("PTSIP_APP_HOST", "127.0.0.1"), "control plane address")
	port := flags.String("port", env("PTSIP_APP_PORT", "8080"), "control plane port")
	db := flags.String("db", env("PTSIP_APP_DB", "ptsip-control-plane.sqlite3"), "SQLite decision database")
	version := flags.Bool("version", false, "print Tool version")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	if *version {
		fmt.Println("PTSIP App " + toolVersion)
		return nil
	}
	number, err := strconv.Atoi(*port)
	if err != nil || number < 1 || number > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	token, secret := os.Getenv("PTSIP_CONTROL_PLANE_TOKEN"), os.Getenv("PTSIP_GITHUB_WEBHOOK_SECRET")
	if token == "" || secret == "" {
		return fmt.Errorf("PTSIP_CONTROL_PLANE_TOKEN and PTSIP_GITHUB_WEBHOOK_SECRET are required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	github, err := app.NewGitHubAppClient()
	if err != nil {
		return err
	}
	store, err := app.OpenStore(ctx, *db)
	if err != nil {
		return err
	}
	defer store.Close()
	service, err := app.NewService(store, github, nil)
	if err != nil {
		return err
	}
	handler, err := app.NewHandler(service, service, token, secret, toolVersion)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr: net.JoinHostPort(*host, *port), Handler: handler, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 15 * time.Second, WriteTimeout: 2 * time.Minute, IdleTimeout: time.Minute,
	}
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	select {
	case <-ctx.Done():
		timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = server.Shutdown(timeout)
	case err = <-done:
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
	}
	return err
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
