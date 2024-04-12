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
