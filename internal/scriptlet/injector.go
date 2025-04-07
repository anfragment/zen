package scriptlet

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/ZenPrivacy/zen-desktop/internal/hostmatch"
	"github.com/ZenPrivacy/zen-desktop/internal/htmlrewrite"
	"github.com/ZenPrivacy/zen-desktop/internal/logger"
)

var (
	//go:embed bundle.js
	defaultScriptletsBundle []byte
	scriptOpeningTag        = []byte("<script>")
	scriptClosingTag        = []byte("</script>")
)

type store interface {
	AddPrimaryRule(hostnamePatterns string, body argList) error
	AddExceptionRule(hostnamePatterns string, body argList) error
	Get(hostname string) []argList
}

// Injector injects scriptlets into HTML HTTP responses.
type Injector struct {
	// bundle contains the <script> element for the scriptlets bundle, which is to be inserted into HTML documents.
	bundle []byte
	// store stores and retrieves scriptlets by hostname.
	store store
}

func NewInjectorWithDefaults() (*Injector, error) {
	store := hostmatch.NewHostMatcher[argList]()
	return newInjector(defaultScriptletsBundle, store)
}

// newInjector creates a new Injector with the embedded scriptlets.
func newInjector(bundleData []byte, store store) (*Injector, error) {
	if bundleData == nil {
		return nil, errors.New("bundleData is nil")
	}
	if store == nil {
		return nil, errors.New("store is nil")
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
	argLists := inj.store.Get(hostname)
	log.Printf("got %d scriptlets for %q", len(argLists), logger.Redacted(hostname))
	if len(argLists) == 0 {
		return nil
	}
	var ruleInjection bytes.Buffer
	ruleInjection.Write(scriptOpeningTag)
	ruleInjection.WriteString("(()=>{")
	var err error
	for _, argList := range argLists {
		if err = argList.GenerateInjection(&ruleInjection); err != nil {
			return fmt.Errorf("generate injection for scriptlet %q: %w", argList, err)
		}
	}
	ruleInjection.WriteString("})();")
	ruleInjection.Write(scriptClosingTag)

	// Appending the scriptlets bundle to the head of the document aligns with the behavior of uBlock Origin:
	// - https://github.com/gorhill/uBlock/blob/d7ae3a185eddeae0f12d07149c1f0ddd11fd0c47/platform/firefox/vapi-background-ext.js#L373-L375
	// - https://github.com/gorhill/uBlock/blob/d7ae3a185eddeae0f12d07149c1f0ddd11fd0c47/platform/chromium/vapi-background-ext.js#L223-L226
	if err := htmlrewrite.AppendHeadContents(res, bytes.Join([][]byte{inj.bundle, ruleInjection.Bytes()}, nil)); err != nil {
		return fmt.Errorf("append head contents: %w", err)
	}

	return nil
}
