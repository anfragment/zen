package rule

import (
	"net/http"
	"testing"
)

func TestRemoveHeaderModifier(t *testing.T) {
	t.Parallel()

	t.Run("returns error if input is invalid", func(t *testing.T) {
		t.Parallel()

		rm := &removeHeaderModifier{}
		if err := rm.Parse("notremoveheader"); err == nil {
			t.Error("expected error to be non-nil")
		}
	})

	t.Run("removes request header", func(t *testing.T) {
		t.Parallel()

		rm := &removeHeaderModifier{}
		if err := rm.Parse("removeheader=request:Authorization"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		req := &http.Request{Header: http.Header{"Authorization": []string{"value"}}}
		if !rm.ModifyReq(req) {
			t.Error("expected request to be modified")
		}

		if req.Header.Get("Authorization") != "" {
			t.Error("expected request header to be removed")
		}
	})

	t.Run("doesn't remove request header if it doesn't exist", func(t *testing.T) {
		t.Parallel()

		rm := &removeHeaderModifier{}
		if err := rm.Parse("removeheader=request:Authorization"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		req := &http.Request{Header: http.Header{}}
		if rm.ModifyReq(req) {
			t.Error("expected request to not be modified")
		}
	})

	t.Run("removes response header", func(t *testing.T) {
		t.Parallel()

		rm := &removeHeaderModifier{}
		if err := rm.Parse("removeheader=Refresh"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		res := &http.Response{Header: http.Header{"Refresh": []string{"value"}}}
		if !rm.ModifyRes(res) {
			t.Error("expected response to be modified")
		}

		if res.Header.Get("Refresh") != "" {
			t.Error("expected response header to be removed")
		}
	})

	t.Run("doesn't remove response header if it doesn't exist", func(t *testing.T) {
		t.Parallel()

		rm := &removeHeaderModifier{}
		if err := rm.Parse("removeheader=Refresh"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		res := &http.Response{Header: http.Header{}}
		if rm.ModifyRes(res) {
			t.Error("expected response to not be modified")
		}
	})
}
