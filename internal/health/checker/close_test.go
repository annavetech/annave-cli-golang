// Copyright 2026 Anna Veretennykova
//
// SPDX-License-Identifier: Apache-2.0
package checker

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"annave.tech/cli/internal/health/domain"
)

var errClose = errors.New("close failed")

// failCloseBody is a response body whose Close fails.
type failCloseBody struct{ io.Reader }

func (failCloseBody) Close() error { return errClose }

// bodyTransport answers every request with status and a body whose Close fails.
type bodyTransport struct{ status int }

func (t bodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: t.status,
		Body:       failCloseBody{strings.NewReader("ok")},
		Header:     http.Header{},
		Request:    req,
	}, nil
}

func TestCheckHTTPReturnsCloseError(t *testing.T) {
	for _, status := range []int{200, 500} {
		target := domain.ServiceTarget{Name: "svc", Address: "http://example.invalid", Timeout: time.Second}
		res, err := checkHTTPWith(context.Background(), target, bodyTransport{status: status})
		if !errors.Is(err, errClose) {
			t.Fatalf("status %d: err = %v, want the close error", status, err)
		}
		want := domain.StatusUp
		if status != 200 {
			want = domain.StatusDown
		}
		if res.Status != want {
			t.Fatalf("status %d: result status = %s, want %s", status, res.Status, want)
		}
	}
}

// failCloseConn is a connection whose Close fails.
type failCloseConn struct{ net.Conn }

func (failCloseConn) Close() error { return errClose }

func TestCheckTCPReturnsCloseError(t *testing.T) {
	dial := func(context.Context, string, string) (net.Conn, error) {
		c1, c2 := net.Pipe()
		t.Cleanup(func() {
			if err := c1.Close(); err != nil {
				t.Error(err)
			}
			if err := c2.Close(); err != nil {
				t.Error(err)
			}
		})
		return failCloseConn{c1}, nil
	}
	target := domain.ServiceTarget{Name: "db", Address: "db:5432", Timeout: time.Second}
	res, err := checkTCPWith(context.Background(), target, dial)
	if !errors.Is(err, errClose) {
		t.Fatalf("err = %v, want the close error", err)
	}
	if res.Status != domain.StatusUp {
		t.Fatalf("status = %s, want up", res.Status)
	}
}

func TestCheckTCPDialFailureHasNoError(t *testing.T) {
	dial := func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("connection refused")
	}
	res, err := checkTCPWith(context.Background(), domain.ServiceTarget{Address: "db:5432"}, dial)
	if err != nil || res.Status != domain.StatusDown {
		t.Fatalf("status = %s, err = %v; want down and no error", res.Status, err)
	}
}
