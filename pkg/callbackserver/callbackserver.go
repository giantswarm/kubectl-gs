package callbackserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/giantswarm/microerror"
)

type CallbackFunc func(http.ResponseWriter, *http.Request) (interface{}, error)

type Config struct {
	Host              string // empty to listen on all network interfaces
	Port              int
	RedirectURI       string
	ReadHeaderTimeout time.Duration
}

type CallbackServer struct {
	host              string
	port              int
	redirectURI       string
	readHeaderTimeout time.Duration
}

// FallthroughResult is an intermediate result returned by the callback function that should be skipped
type FallthroughResult struct {
	Method string
}

func New(config Config) (*CallbackServer, error) {
	var err error

	if config.Port == 0 {
		config.Port, err = findAvailablePort()
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	if len(config.RedirectURI) < 1 {
		config.RedirectURI = "/"
	}

	cs := &CallbackServer{
		host:              config.Host,
		port:              config.Port,
		redirectURI:       config.RedirectURI,
		readHeaderTimeout: config.ReadHeaderTimeout,
	}

	return cs, nil
}

// Run starts a server listening at a given path and port and
// calls a callback function as soon as that path is hit.
//
// It blocks and waits until the given path is hit, or until the
// context deadline is reached, then shuts down the server and
// returns the result of the callback function.
func (cs *CallbackServer) Run(ctx context.Context, callback CallbackFunc) (interface{}, error) {
	resultCh := make(chan interface{})
	errorCh := make(chan error)

	var server *http.Server
	{
		mux := http.NewServeMux()
		mux.HandleFunc(cs.redirectURI, func(w http.ResponseWriter, r *http.Request) {
			result, err := callback(w, r)
			if err != nil {
				errorCh <- err
				return
			}

			if _, ok := result.(FallthroughResult); ok {
				return
			}

			resultCh <- result
		})

		server = &http.Server{
			Addr:              fmt.Sprintf("%s:%d", cs.host, cs.port),
			Handler:           mux,
			ReadHeaderTimeout: cs.readHeaderTimeout,
		}
	}

	go func() {
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			// All good.
		} else if err != nil {
			errorCh <- err
		}
	}()

	var result interface{}
	var origErr error
	select {
	case result = <-resultCh:
		break
	case origErr = <-errorCh:
		break
	case <-ctx.Done():
		origErr = microerror.Mask(timedOutError)
		break
	}

	err := server.Shutdown(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if origErr != nil {
		return nil, microerror.Mask(origErr)
	}

	return result, nil
}

func (cs *CallbackServer) Port() int {
	return cs.port
}

func findAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return -1, microerror.Mask(err)
	}

	ln, err := net.Listen("tcp", addr.String())
	if err != nil {
		return -1, microerror.Mask(err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	return port, nil
}
