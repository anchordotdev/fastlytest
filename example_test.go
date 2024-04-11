package fastlytest

import (
	"context"
	"testing"

	"github.com/fastly/compute-sdk-go/fsthttp"
	"github.com/fastly/compute-sdk-go/fsttest"
)

func TestPong(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hdlVia := fsthttp.HandlerFunc(func(ctx context.Context, w fsthttp.ResponseWriter, r *fsthttp.Request) {
		res, err := r.Send(ctx, "pong")
		if err != nil {
			fsthttp.Error(w, err.Error(), 500)
			return
		}

		w.Header().Reset(res.Header)
		w.Header().Add("Via", "1.1 viceroy-test-vm")

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
	if want, got := "1.1 viceroy-test-vm", w.HeaderMap.Get("Via"); want != got {
		t.Errorf("want via header %q, got %q", want, got)
	}
}
