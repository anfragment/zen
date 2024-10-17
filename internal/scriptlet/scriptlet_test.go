package scriptlet_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/anfragment/zen/internal/scriptlet"
	"github.com/anfragment/zen/internal/scriptlet/triestore"
	"golang.org/x/net/html"
)

func TestInjector(t *testing.T) {
	t.Parallel()

	t.Run("makes an HTML-standards compliant injection", func(t *testing.T) {
		i := newInjectorWithTrieStore(t)
		err := i.AddRule(`#%#//scriptlet('prevent-xhr', 'example.com')`)
		if err != nil {
			t.Fatalf("failed to add rule: %v", err)
		}

		req, err := http.NewRequest("GET", "example.com", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		res := newBlankHttpResponse(t)

		if err := i.Inject(req, res); err != nil {
			t.Errorf("failed to inject: %v", err)
		}

		doc, err := html.Parse(res.Body)
		if err != nil {
			t.Errorf("failed to parse response body after injection: %v", err)
		}

		var metScriptTag bool
		nodeStack := []*html.Node{doc}
		var currNode *html.Node
		for len(nodeStack) > 0 {
			currNode = nodeStack[len(nodeStack)-1]
			nodeStack = nodeStack[:len(nodeStack)-1]
			if currNode.Type == html.ElementNode && currNode.Data == "script" {
				metScriptTag = true
				break
			}

			for c := currNode.FirstChild; c != nil; c = c.NextSibling {
				nodeStack = append(nodeStack, c)
			}
		}

		if !metScriptTag {
			t.Error("expected response body to contain at least one <script> tag, got 0")
		}
	})
}

func newBlankHttpResponse(t *testing.T) *http.Response {
	t.Helper()
	body := io.NopCloser(strings.NewReader(`<html><body></body></html>`))
	header := http.Header{
		"Content-Type": []string{"text/html; charset=UTF-8"},
	}
	return &http.Response{
		Body:   body,
		Header: header,
	}
}

func newInjectorWithTrieStore(t *testing.T) *scriptlet.Injector {
	t.Helper()
	store := triestore.NewTrieStore()
	injector, err := scriptlet.NewInjector(store)
	if err != nil {
		t.Fatalf("failed to create injector: %v", err)
	}
	return injector
}
