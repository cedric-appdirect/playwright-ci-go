package playwrightcigo

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/elazarl/goproxy"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type config struct {
	ctx     context.Context
	timeout time.Duration
}

func New(version string, opts ...Option) (*container, error) {
	c := &config{
		timeout: 60 * time.Second,
		ctx:     context.Background(),
	}
	for _, opt := range opts {
		opt.apply(c)
	}

	ctx, cancel := signal.NotifyContext(c.ctx, os.Interrupt)
	ctx, _ = context.WithTimeout(ctx, c.timeout)

	timeoutSecond := int(c.timeout.Seconds())

	proxy, proxyPort, close := transparentProxy()

	log.Println("Starting and building browser container", fmt.Sprintf("playwright-ci:v%s", version))
	genericContainerReq := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    filepath.Join(".", "docker"),
				Dockerfile: "playwright.Dockerfile",
				Tag:        version,
				Repo:       "playwright-ci",
				KeepImage:  true,
				BuildArgs: map[string]*string{
					"PLAYWRIGHT_VERSION": &version,
				},
				PrintBuildLog: true,
			},
			HostAccessPorts: []int{int(proxyPort)},
			WorkingDir:      "/src",
			ExposedPorts:    []string{"1010/tcp", "1011/tcp", "1012/tcp"},
			Cmd:             []string{fmt.Sprintf("sleep %v", timeoutSecond+10)},
			WaitingFor:      wait.ForExec([]string{"echo", "ready"}),
		},
		Started: true,
	}

	browserContainer, err := testcontainers.GenericContainer(ctx, genericContainerReq)
	if err != nil {
		log.Fatalf("Could not start browser container: %s", err)
	}

	execCtx, execCancel := context.WithCancel(ctx)
	go func() {
		code, _, err := browserContainer.Exec(execCtx, []string{"node", "remote-playwright.js", proxy, strconv.Itoa(proxyPort)})

		// Check that the context is not expired
		select {
		case <-execCtx.Done():
			return
		default:
		}

		if err != nil {
			log.Fatalf("Could not exec in browser container: %s", err)
		}
		if code != 0 {
			log.Fatalf("Exec failed in browser container: %d", code)
		}
	}()

	host, err := browserContainer.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get browser host: %w", err)
	}

	portChromium, err := port(ctx, browserContainer, host, 1010)
	if err != nil {
		return nil, fmt.Errorf("could not get chromium port: %w", err)
	}

	portFirefox, err := port(ctx, browserContainer, host, 1011)
	if err != nil {
		return nil, fmt.Errorf("could not get firefox port: %w", err)
	}

	portWebkit, err := port(ctx, browserContainer, host, 1012)
	if err != nil {
		return nil, fmt.Errorf("could not get webkit port: %w", err)
	}

	return &container{
		host: host,
		ports: struct {
			chromium int
			firefox  int
			webkit   int
		}{
			chromium: portChromium,
			firefox:  portFirefox,
			webkit:   portWebkit,
		},
		terminate: func() {
			execCancel()
			if err := browserContainer.Terminate(ctx); err != nil {
				log.Fatalf("Could not purge browser container: %s", err)
			}
			close()
			cancel()
		},
	}, nil
}

func transparentProxy() (string, int, func()) {
	// Listen for incoming connections
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("Error listening:", err)
	}

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	srv := &http.Server{
		Handler:           proxy,
		ReadHeaderTimeout: time.Second * 5, // Set a reasonable ReadHeaderTimeout value
	}

	go func() {
		srv.Serve(l)
	}()

	split := strings.Split(l.Addr().String(), ":")
	port, _ := strconv.ParseInt(split[1], 10, 64)

	if err := wait4port("http://" + l.Addr().String()); err != nil {
		log.Fatalf("Could not connect to proxy: %s", err)
	}

	return "http://" + testcontainers.HostInternal + ":" + split[1], int(port), func() {
		srv.Shutdown(context.Background())
		l.Close()
	}
}

func wait4port(addr string) error {
	time.Sleep(time.Second)
	for i := 0; i < 15; i++ {
		resp, err := http.Get(addr)
		if err != nil {
			log.Println("could not connect to", addr, "error", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if err := resp.Body.Close(); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("could not connect to %s after retry and timeout", addr)
}

func port(ctx context.Context, container testcontainers.Container, host string, port int) (int, error) {
	p, err := container.MappedPort(ctx, nat.Port(fmt.Sprintf("%d/tcp", port)))
	if err != nil {
		return 0, fmt.Errorf("could not get browser port: %w", err)
	}
	if err := wait4port(fmt.Sprintf("http://%s:%d", host, p.Int())); err != nil {
		return 0, fmt.Errorf("timeout, could not connect to browser container: %w", err)
	}
	return p.Int(), nil
}
