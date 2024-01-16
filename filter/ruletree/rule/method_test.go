package rule

import (
	"net/http"
	"testing"
)

func TestSingleMethod(t *testing.T) {
	t.Parallel()

	m := methodModifier{}
	m.Parse("method=GET")
	req := http.Request{
		Method: "GET",
	}
	if !m.ShouldMatch(&req) {
		t.Error("method=GET should match a GET request")
	}
}

func TestSingleInvertedMethod(t *testing.T) {
	t.Parallel()

	m := methodModifier{}
	m.Parse("method=~GET")
	req := http.Request{
		Method: "GET",
	}
	if m.ShouldMatch(&req) {
		t.Error("method=~GET should not match a GET request")
	}
}

func TestLowercaseMethod(t *testing.T) {
	t.Parallel()

	m := methodModifier{}
	m.Parse("method=get")
	req := http.Request{
		Method: "GET",
	}
	if !m.ShouldMatch(&req) {
		t.Error("method=get should match a GET request")
	}
}

func TestMultipleMethods(t *testing.T) {
	t.Parallel()

	m := methodModifier{}
	m.Parse("method=GET|POST")

	req := http.Request{
		Method: "GET",
	}
	if !m.ShouldMatch(&req) {
		t.Error("method=GET|POST should match a GET request")
	}

	req.Method = "POST"
	if !m.ShouldMatch(&req) {
		t.Error("method=GET|POST should match a POST request")
	}

	req.Method = "HEAD"
	if m.ShouldMatch(&req) {
		t.Error("method=GET|POST should not match a HEAD request")
	}
}

func TestMultipleInvertedMethods(t *testing.T) {
	t.Parallel()

	m := methodModifier{}
	m.Parse("method=~GET|~POST")

	req := http.Request{
		Method: "GET",
	}
	if m.ShouldMatch(&req) {
		t.Error("method=~GET|~POST should not match a GET request")
	}

	req.Method = "POST"
	if m.ShouldMatch(&req) {
		t.Error("method=~GET|~POST should not match a POST request")
	}

	req.Method = "HEAD"
	if !m.ShouldMatch(&req) {
		t.Error("method=~GET|~POST should match a HEAD request")
	}

	req.Method = "PUT"
	if !m.ShouldMatch(&req) {
		t.Error("method=~GET|~POST should match a PUT request")
	}
}

func TestMixedInvertedAndNonInvertedMethods(t *testing.T) {
	t.Parallel()

	m := methodModifier{}
	if err := m.Parse("method=GET|~POST"); err == nil {
		t.Error("method=GET|~POST should return an error")
	}

	m = methodModifier{}
	if err := m.Parse("method=~GET|POST"); err == nil {
		t.Error("method=~GET|POST should return an error")
	}
}
