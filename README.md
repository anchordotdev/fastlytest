# fastlytest

A "go test" executor for [Viceroy](https://github.com/fastly/Viceroy). Execute
tests that run both inside a WASM VM and on the host system.

## Setup

[Install standalone viceroy on your
system](https://github.com/fastly/Viceroy?tab=readme-ov-file#as-a-standalone-tool-from-cratesio),
and add it to `PATH` for the `go test` command. Add a `TestMain` func that
initializes a `Config` and runs starts a `Viceroy` instance:

sample `main_test.go`

``` go
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
```

Add a test that sends a `fsthttp.Request` to the edge handler, which can proxy
to the server on the host:

sample `compute_test.go`:

``` go
package fastlytest

import (
	"context"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
)

func TestVia(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const hdrVia = "1.1 viceroy-test-vm"

	hdlVia := fsthttp.HandlerFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		res, err := r.Send(ctx, "test-via")
		if err != nil {
			fsthttp.Error(w, err.Error(), 500)
			return
		}

		w.Header().Reset(res.Header)
		w.Header().Add("Via", hdrVia)

		w.WriteHeader(res.StatusCode)
	})

	r, err := fsthttp.NewRequest("GET", "http://pong.example.test/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := fsttest.NewRecorder()

	hdlVia(ctx, w, r)

	if want, got := fsthttp.StatusOK, w.Code; want != got {
		t.Errorf("want status code %d, got %d", want, got)
	}

	// assert the header set in hdlVia

	if want, got := hdrVia, w.HeaderMap.Get("Via"); want != got {
		t.Errorf("want via header %q, got %q", want, got)
	}

	// assert the header set in srvCustom

	if want, got := "httptest-server", w.HeaderMap.Get("Server"); want != got {
		t.Errorf("want server header %q, got %q", want, got)
	}
}
```


### Test Process Hierarchy

When `go test` is run on the host, a child process is created which re-executes
the `go test` command for WASM using viceroy as the program runner. The tests
execute inside of the WASM VM.

``` mermaid
flowchart
    parent[parent #quot;go test#quot; process]

    child[child #quot;go test -exec viceroy#quot; process]

    subgraph viceroy[viceroy #quot;run pkg.test#quot; process]
        wasm["wasm #quot;go test#quot; process"]
    end

    parent-->child
    child-->viceroy
```

### Test Request/Response Flow

Requests originate from the `TestFunc` inside the WASM VM, and are handled
first by the `fsthttp.

``` mermaid
sequenceDiagram
    box WASM
        participant TestFunc as Test Func
        participant TestHandler as Test HTTP Handler
    end

    box Host
        participant Viceroy as Viceroy Reverse Proxy
        participant TestMain as TestMain External Server
    end

    TestFunc->>+TestHandler: test request (fsthttp.Request)
    TestHandler->>+Viceroy: forward request (req.Send)
    Viceroy->>+TestMain: proxy request
    TestMain->>+Viceroy: send response
    Viceroy->>+TestHandler: proxy response (fsthttp.Response)
    TestHandler->>+TestFunc: test response (fsthttp.ResponseWriter)
```
