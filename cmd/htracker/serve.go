package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oklog/run"
	httptransport "gitlab.com/henri.philipps/htracker/http"
	"gitlab.com/henri.philipps/htracker/scraper"
	"gitlab.com/henri.philipps/htracker/service"
	"gitlab.com/henri.philipps/htracker/storage/memory"
	"gitlab.com/henri.philipps/htracker/storage/postgres"
	"gitlab.com/henri.philipps/htracker/watcher"
	"golang.org/x/exp/slog"
)

const memoryBackend = "memory"
const postgresBackend = "postgres"

var (
	servefs         = flag.NewFlagSet("serve", flag.ExitOnError)
	addrFlag        = servefs.String("addr", ":8080", "address the server is listening on")
	chromeWSFlag    = servefs.String("ws", "ws://localhost:3000", "websocket url of chrome instance to connect to for site rendering")
	intervalFlag    = servefs.Int("interval", 3600, "interval in seconds between watcher runs")
	gracePeriodFlag = servefs.Int("grace", 10, "shutdown grace period in seconds")
	backendFlag     = servefs.String("backend", memoryBackend, "the storage backend (memory|postgres)")
	postgresFlag    = servefs.String("pguri", "postgres://localhost?sslmode=disable", "postgres connection uri")
)

// newServeFunc creates the func which is executed by servecmd.
func newServeFunc() func(context.Context, []string) error {

	return func(serveCtx context.Context, args []string) error {
		ctx, cancel := context.WithCancel(serveCtx)
		logger, err := createLogger(*logLevelFlag)
		if err != nil {
			return err
		}

		var archive service.SiteArchive
		var subscriptionSvc service.SubscriptionSvc

		switch *backendFlag {
		case memoryBackend:
			archive = service.NewSiteArchive(memory.NewSiteStorage(logger))
			subscriptionSvc = service.NewSubscriptionSvc(memory.NewSubscriptionStorage(logger))
		case postgresBackend:
			storage, err := postgres.New(*postgresFlag, logger)
			if err != nil {
				return err
			}
			archive = service.NewSiteArchive(storage)
			subscriptionSvc = service.NewSubscriptionSvc(storage, service.WithLogger(logger))
		default:
			return fmt.Errorf("storage backend %s not supported", *backendFlag)
		}

		watcherOpts := []watcher.Opt{
			watcher.WithInterval(time.Duration(*intervalFlag) * time.Second),
			watcher.WithLogger(logger),
		}

		if *chromeWSFlag != "" {
			watcherOpts = append(watcherOpts, watcher.WithScraperOpts(scraper.WithBrowserEndpoint(*chromeWSFlag)))
		}

		watcher := watcher.NewWatcher(archive, subscriptionSvc, watcherOpts...)
		router := httptransport.MakeAPIHandler(archive, subscriptionSvc, logger)

		// the run group will take care of running and shutting down all background components
		g := run.Group{}

		// add handler for signals to run group, for shutting down all components on SIGINT and SIGTERM
		g.Add(func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-c:
				return fmt.Errorf("catched signal %v", sig)
			case <-ctx.Done():
				return ctx.Err()
			}
		}, func(error) {
			cancel()
		})

		// add watcher to run group
		g.Add(func() error { return watcher.Start(ctx) }, func(error) { cancel() })

		// Instead of ListenAndServe(), which can't be interrupted, we create our own
		// Server and add it's Serve() method to the run group later.
		// We set ReadHeaderTimeout to prevent Slowloris attacks.
		server := http.Server{Handler: router, ReadHeaderTimeout: 5 * time.Second}

		logger.Info("start listening...", slog.String("listen_addr", *addrFlag))
		ln, err := net.Listen("tcp", *addrFlag)
		if err != nil {
			logger.Error("failed to start server, exiting", err)
			return err
		}

		graceTimeoutCtx, _ := context.WithTimeout(ctx, time.Duration(*gracePeriodFlag)*time.Second)
		g.Add(func() error { return server.Serve(ln) }, func(error) {
			if err := server.Shutdown(graceTimeoutCtx); err != nil {
				logger.Error("graceful shutdown error", err)
			}
		})

		err = g.Run()
		logger.Info("exiting", slog.String("reason", err.Error()))
		return err
	}
}
