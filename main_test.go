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
	// wrap os.Exit in a function call so that defer's fire before process exit.

	os.Exit(func() int {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// create a test server for each handler to test, this one adds a custom
		// "Server" header to the response

		srvCustom := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Server", "httptest-server")
		}))
		defer srvCustom.Close()

		// create fastly config, add a backend for each test server or test case

		cfg := Config{
			LocalServer: LocalServer{
				Backends: map[string]Backend{
					"test-via": {URL: srvCustom.URL},
				},
			},
		}

		// create viceroy runner, set the config

		vic, err := NewViceroy(cfg)
		if err != nil {
			panic(err)
		}
		defer vic.Cleanup()

		// execute the go test command for this package via viceroy

		if err = vic.GoTestPkg(ctx, "fastlytest").Run(); err == nil {
			return 0
		}

		// exit with the same code after the tests have run via viceroy to indicate pass/fail

		var eerr *exec.ExitError
		if errors.Is(err, eerr) {
			return eerr.ProcessState.ExitCode()
		}
		return -1
	}())
}
