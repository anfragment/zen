package scriptlet

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/anfragment/zen/internal/htmlrewrite"
	"github.com/anfragment/zen/internal/logger"
)

var (
	//go:embed bundle.js
	scriptletsBundleFS embed.FS
	// reBody captures contents of the body tag in an HTML document.
	reBody           = regexp.MustCompile(`(?i)<body[\s\S]*?>([\s\S]*)</body>`)
	scriptOpeningTag = []byte("<script>")
	scriptClosingTag = []byte("</script>")
)

type Store interface {
	Add(hostnames []string, scriptlet *Scriptlet)
	Get(hostname string) []*Scriptlet
}

// Injector injects scriptlets into HTML HTTP responses.
type Injector struct {
	// bundle contains the <script> element for the scriptlets bundle, which is to be inserted into HTML documents.
	bundle []byte
	// store stores and retrieves scriptlets by hostname.
	store Store
}

// NewInjector creates a new Injector with the embedded scriptlets.
func NewInjector(store Store) (*Injector, error) {
	if store == nil {
		return nil, errors.New("store is nil")
	}

	bundleData, err := scriptletsBundleFS.ReadFile("bundle.js")
	if err != nil {
		return nil, fmt.Errorf("read bundle from embed: %w", err)
	}

	scriptletsElement := make([]byte, len(scriptOpeningTag)+len(bundleData)+len(scriptClosingTag))
	copy(scriptletsElement, scriptOpeningTag)
	copy(scriptletsElement[len(scriptOpeningTag):], bundleData)
	copy(scriptletsElement[len(scriptOpeningTag)+len(bundleData):], scriptClosingTag)

	return &Injector{
		bundle: scriptletsElement,
		store:  store,
	}, nil
}

// Inject injects scriptlets into a given HTTP HTML response.
//
// On error, the caller may proceed as if the function had not been called.
func (inj *Injector) Inject(req *http.Request, res *http.Response) error {
	hostname := req.URL.Hostname()
	scriptlets := inj.store.Get(hostname)
	log.Printf("got %d scriptlets for %q", len(scriptlets), logger.Redacted(hostname))
	if len(scriptlets) == 0 {
		return nil
	}
	var ruleInjection bytes.Buffer
	ruleInjection.Write(scriptOpeningTag)
	ruleInjection.WriteString("\n(function() {\n")
	var err error
	for _, scriptlet := range scriptlets {
		if err = scriptlet.GenerateInjection(&ruleInjection); err != nil {
			return fmt.Errorf("generate injection for scriptlet %q: %w", scriptlet.Name, err)
		}
		ruleInjection.WriteByte('\n')
	}
	ruleInjection.WriteString("})();\n")
	ruleInjection.Write(scriptClosingTag)

	htmlrewrite.ReplaceBodyContents(res, func(match []byte) []byte {
		match = append(match, inj.bundle...)
		match = append(match, ruleInjection.Bytes()...)
		return match
	})

	return nil
}
