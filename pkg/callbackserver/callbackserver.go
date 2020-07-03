package callbackserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/giantswarm/microerror"
)

// CallbackResult is used by our channel to store callback results.
type CallbackResult struct {
	Interface interface{}
	Error     error
}

// RunCallbackServer starts a server listening at a specific path and port and
// calls a callback function as soon as that path is hit.
//
// It blocks and waits until the path is hit, then shuts down and returns the
// result of the callback function.
func RunCallbackServer(port int, redirectURI string, callback func(w http.ResponseWriter, r *http.Request) (interface{}, error)) (interface{}, error) {
	// Set a channel we will block on and wait for the result.
	resultCh := make(chan CallbackResult)

	// Setup the server.
	m := http.NewServeMux()
	s := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: m}

	// This is the handler for the path we specified, it calls the provided
	// callback as soon as a request arrives and moves the result of the callback
	// on to the resultCh.
	m.HandleFunc(redirectURI, func(w http.ResponseWriter, r *http.Request) {
		// Got a response, call the callback function.
		i, err := callback(w, r)
		resultCh <- CallbackResult{i, err}
	})

	// Start the server.
	go startServer(s)

	// Block till the callback gives us a result.
	r := <-resultCh

	// Shutdown the server.
	err := s.Shutdown(context.Background())
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Return the result.
	return r.Interface, r.Error
}

func startServer(s *http.Server) {
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
