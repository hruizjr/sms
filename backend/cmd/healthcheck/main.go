// Command healthcheck is the container health check of the backend image (Phase 3 Group 6, owner
// decision L3). The runtime image (distroless) has no shell, curl, or wget, so Docker runs this
// program inside the api container.
//
// It sends one GET request to the API's readiness endpoint on the loopback interface and exits 0 when
// the API answers 200 OK, 1 otherwise (not ready, unreachable, timed out, or redirected), and 2 on a
// usage error. It uses only the standard library, follows no redirects, and reads at most 64 KiB of
// the response.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	defaultURL     = "http://127.0.0.1:8080/readyz"
	defaultTimeout = 3 * time.Second
	maxBodyBytes   = 64 << 10

	exitHealthy   = 0
	exitUnhealthy = 1
	exitUsage     = 2
)

func main() {
	os.Exit(run(os.Args[1:], http.DefaultTransport, os.Stderr))
}

// run parses the arguments, probes the URL through transport, and returns the exit code.
func run(args []string, transport http.RoundTripper, stderr io.Writer) int {
	fs := flag.NewFlagSet("healthcheck", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("url", defaultURL, "readiness URL to probe (http, loopback IP address only)")
	timeout := fs.Duration("timeout", defaultTimeout, "time allowed for the whole request")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "healthcheck: unexpected argument %q\n", fs.Arg(0))
		return exitUsage
	}
	if *timeout <= 0 {
		_, _ = fmt.Fprintln(stderr, "healthcheck: -timeout must be positive")
		return exitUsage
	}
	if err := validateURL(*target); err != nil {
		_, _ = fmt.Fprintln(stderr, "healthcheck:", err)
		return exitUsage
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	client := &http.Client{
		Transport: transport,
		// A redirect is not a healthy answer; never follow it.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	if err := probe(ctx, client, *target); err != nil {
		_, _ = fmt.Fprintln(stderr, "healthcheck:", err)
		return exitUnhealthy
	}
	return exitHealthy
}

// validateURL accepts only http URLs whose host is a loopback IP address with an explicit port, so
// the probe can only ever reach a process in the same container or on the same machine.
func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid -url: %w", err)
	}
	if u.Scheme != "http" {
		return errors.New("invalid -url: scheme must be http")
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil || port == "" {
		return errors.New("invalid -url: host must be a loopback IP address with a port")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("invalid -url: host must be a loopback IP address with a port")
	}
	return nil
}

func probe(ctx context.Context, client *http.Client, target string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() // read-only response; a close error changes nothing
	// Drain a bounded amount so the connection closes cleanly; the content is not needed.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("not ready: %s answered %d", target, resp.StatusCode)
	}
	return nil
}
