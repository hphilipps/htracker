package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/oklog/run"
	"gitlab.com/henri.philipps/htracker"
	"gitlab.com/henri.philipps/htracker/endpoint"
	"gitlab.com/henri.philipps/htracker/exporter"
	httptransport "gitlab.com/henri.philipps/htracker/http"
	"gitlab.com/henri.philipps/htracker/scraper"
	"gitlab.com/henri.philipps/htracker/service"
	"gitlab.com/henri.philipps/htracker/storage/memory"
	"gitlab.com/henri.philipps/htracker/watcher"
	"golang.org/x/exp/slog"
)

var servefs = flag.NewFlagSet("serve", flag.ExitOnError)
var scrapefs = flag.NewFlagSet("scrape", flag.ExitOnError)
var watchfs = flag.NewFlagSet("watch", flag.ExitOnError)

var addrFlag = servefs.String("addr", ":8080", "address the server is listening on")

var urlFlag = scrapefs.String("url", "", "url to be scraped")
var filterFlag = scrapefs.String("f", "", "filter to be applied to scraped content")
var contentTypeFlag = scrapefs.String("t", "text", "content type of the scraped url")
var chromeWSFlag = scrapefs.String("ws", "ws://localhost:3000", "websocket url to connect to chrome instance for site rendering")
var renderWithChromeFlag = scrapefs.Bool("r", false, "render site content using chrome if true (needed for java script)")

func main() {
	ctx := context.Background()
	logger := slog.Default()
	storage := memory.NewSiteStorage(logger)
	archive := service.NewSiteArchive(storage)
	subscriptionSvc := service.NewSubscriptionSvc(storage)

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "scrape":
		if err := scrapefs.Parse(os.Args[2:]); err != nil {
			fmt.Println(err.Error())
			os.Exit(0)
		}
		scrape(ctx, *urlFlag, *filterFlag, *contentTypeFlag, archive, *logger)
	case "serve":
		if err := servefs.Parse(os.Args[2:]); err != nil {
			fmt.Println(err.Error())
			os.Exit(0)
		}
		serve(ctx, *addrFlag, archive, subscriptionSvc, *logger)
	case "watch":
		if err := watchfs.Parse(os.Args[2:]); err != nil {
			fmt.Println(err.Error())
			os.Exit(0)
		}
		watch(ctx, subscriptionSvc, archive, *logger)
	default:
		usage()
	}
}

func usage() {
	fmt.Printf("Usage: %s serve|scrape|watch [options]\n", filepath.Base(os.Args[0]))
}

func serve(ctx context.Context, listenAddr string, archive service.SiteArchive, subscriptionSvc service.SubscriptionSvc, logger slog.Logger) {

	ctx, cancel := context.WithCancel(ctx)

	watcher := watcher.NewWatcher(archive, subscriptionSvc, watcher.WithInterval(60*time.Second), watcher.WithLogger(&logger))
	go watcher.Start(ctx)

	router := httptransport.MakeAPIHandler(archive, subscriptionSvc, &logger)

	g := run.Group{}

	// add handler for signals, to shutdown all go routines on SIGINT and SIGTERM
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

	// instead of ListenAndServe(), which can't be interrupted, we create our own
	// Server and add it's Serve() method to the run group later.
	server := http.Server{Handler: router}

	logger.Info("start listening...", slog.String("listen_addr", listenAddr))
	ln, err := net.Listen("tcp", *addrFlag)
	if err != nil {
		logger.Error("failed to start server, exiting", err)
	}

	timeout, _ := context.WithTimeout(ctx, 30*time.Second)
	g.Add(func() error { return server.Serve(ln) }, func(error) { server.Shutdown(timeout) })
	logger.Info("exiting", slog.String("reason", g.Run().Error()))
}

func createJSONHandler[Req endpoint.Requester, Resp endpoint.Responder](ep endpoint.Endpoint[Req, Resp]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		request, err := decodeHTTPJSONRequest[Req](ctx, req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			errResponse := struct{ Error string }{Error: fmt.Sprintf("request decoder: %s", err.Error())}
			if err := json.NewEncoder(w).Encode(errResponse); err != nil {
				panic(err)
			}
			return
		}

		response, err := ep(ctx, request)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errResponse := struct{ Error string }{Error: err.Error()}
			if err := json.NewEncoder(w).Encode(errResponse); err != nil {
				panic(err)
			}
			return
		}

		if err := encodeHTTPJSONResponse(ctx, w, response); err != nil {
			panic(err)
		}
	}
}

func decodeHTTPJSONRequest[Req endpoint.Requester](_ context.Context, r *http.Request) (Req, error) {
	var req Req
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func encodeHTTPJSONResponse[Resp endpoint.Responder](ctx context.Context, w http.ResponseWriter, response Resp) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := response.Failed(); err != nil {
		switch {
		case errors.Is(err, htracker.ErrNotExist):
			w.WriteHeader(http.StatusNotFound)
		case errors.Is(err, htracker.ErrAlreadyExists):
			w.WriteHeader(http.StatusConflict)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		errResponse := struct{ Error string }{Error: err.Error()}
		return json.NewEncoder(w).Encode(errResponse)
	}
	return json.NewEncoder(w).Encode(response)
}

func scrape(ctx context.Context, url, filter, contentType string, archive service.SiteArchive, logger slog.Logger) {
	exp := exporter.NewExporter(context.Background(), archive)
	subscription := &htracker.Subscription{
		URL:         *urlFlag,
		Filter:      *filterFlag,
		ContentType: *contentTypeFlag,
	}

	opts := []scraper.Opt{scraper.WithExporters([]exporter.Interface{exp})}
	if *renderWithChromeFlag {
		opts = append(opts, scraper.WithBrowserEndpoint(*chromeWSFlag))
	}

	h1 := scraper.NewScraper(
		[]*htracker.Subscription{subscription},
		opts...,
	)

	h2 := scraper.NewScraper(
		[]*htracker.Subscription{subscription},
		opts...,
	)

	h1.Start()

	site, err := archive.Get(ctx, subscription)
	if err != nil {
		logger.Error("db.Get() failed", err)
		os.Exit(1)
	}

	fmt.Printf("Site on 1st update: lastUdpdated: %v, lastChecked: %v, checksum: %s, diff: %s\ncontent: %s\n", site.LastUpdated, site.LastChecked, site.Checksum, site.Diff, site.Content)
	content1 := site.Content

	time.Sleep(time.Second)

	h2.Start()

	site, err = archive.Get(ctx, subscription)
	if err != nil {
		logger.Error("svc.Get() failed", err)
		os.Exit(1)
	}

	fmt.Printf("Site on 2nd update: lastUdpdated: %v, lastChecked: %v, checksum: %s, diff: --%s--\ncontent: %s\n", site.LastUpdated, site.LastChecked, site.Checksum, site.Diff, site.Content)

	fmt.Println("len c1:", len(content1), "len c2:", len(site.Content))

	for i, c := range content1 {
		if c != site.Content[i] {
			fmt.Println(i, ": ", c, "!=", site.Content[i])
			fmt.Println(string(content1)[i-5 : i+10])
			fmt.Println(string(site.Content)[i-5 : i+10])

			break
		}
	}
}

func watch(ctx context.Context, subscriptionSvc service.SubscriptionSvc, archive service.SiteArchive, logger slog.Logger) {
	subscriptionSvc.AddSubscriber(ctx, &service.Subscriber{Email: "email1"})
	subscriptionSvc.AddSubscriber(ctx, &service.Subscriber{Email: "email2"})
	subscriptionSvc.Subscribe(ctx, "email1", &htracker.Subscription{URL: "http://httpbin.org/anything/1"})
	subscriptionSvc.Subscribe(ctx, "email1", &htracker.Subscription{URL: "http://httpbin.org/anything/2"})
	subscriptionSvc.Subscribe(ctx, "email2", &htracker.Subscription{URL: "http://httpbin.org/anything/2"})

	dbgLogger := slog.New(slog.HandlerOptions{Level: slog.LevelDebug}.NewTextHandler(os.Stdout))

	w := watcher.NewWatcher(archive, subscriptionSvc, watcher.WithInterval(5*time.Second), watcher.WithLogger(dbgLogger))

	tctx, _ := context.WithTimeout(ctx, 20*time.Second)
	if err := w.Start(tctx); err != nil {
		logger.Error("Watcher", err)
	}

	sa, err := archive.Get(tctx, &htracker.Subscription{URL: "http://httpbin.org/anything/2"})
	if err != nil {
		logger.Error("ArchiveService", err)
	}

	fmt.Printf("LU: %v: %v\n", sa.LastUpdated, sa.Diff)
}
