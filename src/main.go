package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/alerting"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/daemon"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/http"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/s3"
)

var applicationName string
var buildMode string
var commitSha string

// Pre-setup, runs before main.
func init() {
	flags.SetupFlags(applicationName, buildMode, commitSha)
}

// Main entrypoint.
func main() {
	if *flags.HelpFlag {
		flag.Usage()

		return
	}

	http.Setup()
	alerting.Setup()
	cache.Setup()
	s3.Setup()

	if *flags.Pulse {
		err := daemon.DoWork()
		if err != nil {
			glog.Fatalf("Error during work: %v", err)
		}

		return
	}

	go daemon.Run()

	// Wait for a signal to quit
	daemon.CloseSignal = make(chan os.Signal, 1)

	// We want to catch ALL signals to quit
	signal.Notify(daemon.CloseSignal, syscall.SIGABRT, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		sig := <-daemon.CloseSignal

		glog.Flush()

		close(daemon.CloseSignal)

		glog.Warningf("Received signal %s, exiting\n", sig)

		os.Exit(0)
	}()
}
