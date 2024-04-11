//go:build !wasip1 || nofastlyhostcalls

// ^ important build tags

package fastlytest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start a backend server on the host

	srvPong := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong!"))
	}))
	defer srvPong.Close()

	// create fastly config, add fe-cails backend to it

	cfg := Config{
		LocalServer: LocalServer{
			Backends: map[string]Backend{
				"pong": {
					URL: srvPong.URL,
				},
			},
		},
	}

	// create viceroy runner, set the config

	vic, err := NewViceroy(cfg)
	if err != nil {
		panic(err)
	}
	defer vic.Cleanup()

	// exit with the same code after the tests have run via viceroy to indicate pass/fail

	os.Exit(func() int {
		// execute the go test command for this package via viceroy
		if err = vic.GoTestPkg(ctx, "fastlytest").Run(); err == nil {
			return 0
		}

		var eerr *exec.ExitError
		if errors.Is(err, eerr) {
			return eerr.ProcessState.ExitCode()
		}
		return -1
	}())
}
