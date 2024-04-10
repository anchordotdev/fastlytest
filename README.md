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

// ^ important build tags to prevent TestMain from re-running in the WASM VM

package foo

import "github.com/anchordotdev/fastlytest"

func TestMain(m *testing.M) {
	// wrap the whole func body so that defers still run before exit
	os.Exit(func() int {

		// start a backend server on the host
	
		srv := httptest.NewServer(nil)
		defer srv.Close()
	
		// create fastly config, add fe-cails backend to it
	
		cfg := fastlytest.Config{
			LocalServer: fastlytest.LocalServer{
				Backends: map[string]fastlytest.Backend{
					"test-backend": { URL: srv.URL },
				},
			},
		}
	
		// create viceroy runner, set the config
	
		vic, err := fastlytest.NewViceroy(cfg)
		if err != nil {
			panic(err)
		}
		defer vic.Cleanup()

		// execute the go test command for this package via viceroy
		if err = vic.GoTestPkg(ctx, "fastlytest").Run(); err == nil {
			return 0
		}

		// exit with the same code after the tests have run via viceroy to
		// indicate pass/fail

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
func TestVia(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// edge "middleware" to test

	hdlVia := fsthttp.HandlerFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
        // proxy this request to the httptest.Server running on the host

		res, err := r.Send(ctx, "test-backend") // backend name from the Config
		if err != nil {
			fsthttp.Error(w, err.Error(), 500)
			return
		}

		// Add a 'Via' header to the response

		w.Header().Reset(res.Header)
		w.Header().Add("Via", "1.1 viceroy-test-vm")

		w.WriteHeader(res.StatusCode)
	})

	// build a test request and send it into the handler, and record the response

	r, err := fsthttp.NewRequest("GET", "http://pong.example.test/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := fsttest.NewRecorder()

	hdlVia(ctx, w, r)

	if want, got := fsthttp.StatusOK, w.Code; want != got {
		t.Errorf("want status code %d, got %d", want, got)
	}
	if want, got := "1.1 viceroy-test-vm", w.HeaderMap.Get("Via"); want != got {
		t.Errorf("want via header %q, got %q", want, got)
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
