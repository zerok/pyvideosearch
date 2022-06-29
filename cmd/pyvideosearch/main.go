package main

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/zerok/pyvideosearch/http"
	"github.com/zerok/pyvideosearch/index"

	"runtime"

	"sync"

	"github.com/spf13/pflag"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	var dataFolder string
	var indexPath string
	var addr string
	var forceRebuild bool
	var baseURL string
	var checkInterval time.Duration
	var startHTTPD bool
	allowedOrigins := make([]string, 0, 1)
	pflag.StringVar(&dataFolder, "data-path", "", "Path to the pyvideo data folder")
	pflag.StringVar(&indexPath, "index-path", "search.bleve", "Path to the search index folder")
	pflag.StringVar(&addr, "http-addr", "127.0.0.1:8080", "Address the HTTP server should listen on for API calls")
	pflag.BoolVar(&startHTTPD, "http", false, "Start HTTPD")
	pflag.BoolVar(&forceRebuild, "force-rebuild", false, "Rebuild the index even if it already exists")
	pflag.StringVar(&baseURL, "base-url", "http://pyvideo.org", "Base URL of the pyvideo website")
	pflag.StringSliceVar(&allowedOrigins, "allowed-origin", []string{"http://localhost:8000"}, "(CORS) allowed hostname for XHRs")
	pflag.DurationVar(&checkInterval, "check-interval", 0, "Interval in which the data folder is updated from upstream using git pull")
	pflag.Parse()

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr})

	if dataFolder == "" {
		logger.Fatal().Msg("Please specify the path to the pyvideo data folder using --data-path")
	}

	idxChan := make(chan *index.Index, 1)
	ctx, cancel := context.WithCancel(logger.WithContext(context.Background()))
	defer cancel()

	var mainGrp sync.WaitGroup
	mainGrp.Add(1)
	if startHTTPD {
		mainGrp.Add(1)
	}

	go func() {
		idx, err := index.LoadIndex(ctx, indexPath, dataFolder, forceRebuild, true)
		if err != nil {
			logger.Fatal().Err(err).Msgf("Failed to load index on %s", indexPath)
		}
		idxChan <- idx

		if checkInterval == 0 {
			logger.Info().Msg("Check interval set to 0. Disabling automatic updates.")
			return
		}

		if err := index.WatchForUpdates(ctx, idxChan, indexPath, dataFolder, checkInterval, !startHTTPD); err != nil {
			logger.Fatal().Err(err).Msg("Failed to watch-update data folder")
		}

		mainGrp.Done()
	}()

	if startHTTPD {
		if err := http.RunHTTPD(ctx, idxChan, addr, allowedOrigins); err != nil {
			logger.Fatal().Err(err).Msgf("Failed to start HTTPD on %s", addr)
		}
		mainGrp.Done()
	}

	mainGrp.Wait()
}
