package cosmetic

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/anfragment/zen/internal/hostmatch"
	"github.com/anfragment/zen/internal/htmlrewrite"
	"github.com/anfragment/zen/internal/logger"
)

var (
	RuleRegex          = regexp.MustCompile(`.*#@?#.+`)
	primaryRuleRegex   = regexp.MustCompile(`(.*?)##(.*)`)
	exceptionRuleRegex = regexp.MustCompile(`(.*?)#@#(.+)`)

	injectionStart = []byte("<style>")
	injectionEnd   = []byte("</style>")
)

type store interface {
	AddPrimaryRule(hostnamePatterns string, selector string) error
	AddExceptionRule(hostnamePatterns string, selector string) error
	Get(hostname string) []string
}

type Injector struct {
	store store
}

func NewInjector() *Injector {
	return &Injector{
		store: hostmatch.NewHostMatcher[string](),
	}
}

func (inj *Injector) AddRule(rule string) error {
	if match := primaryRuleRegex.FindStringSubmatch(rule); match != nil {
		if err := inj.store.AddPrimaryRule(match[1], match[2]); err != nil {
			return fmt.Errorf("add primary rule: %w", err)
		}
		return nil
	}

	if match := exceptionRuleRegex.FindStringSubmatch(rule); match != nil {
		if err := inj.store.AddExceptionRule(match[1], match[2]); err != nil {
			return fmt.Errorf("add exception rule: %w", err)
		}
		return nil
	}

	return errors.New("unsupported syntax")
}

func (inj *Injector) Inject(req *http.Request, res *http.Response) error {
	hostname := req.URL.Hostname()
	selectors := inj.store.Get(hostname)
	log.Printf("got %d cosmetic rules for %q", len(selectors), logger.Redacted(hostname))
	if len(selectors) == 0 {
		return nil
	}

	var ruleInjection bytes.Buffer
	ruleInjection.Write(injectionStart)
	css := generateBatchedCSS(selectors)
	ruleInjection.WriteString(css)
	ruleInjection.Write(injectionEnd)

	// Why append and not prepend?
	// When multiple CSS rules define an !important property, conflicts are resolved first by specificity and then by the order of the CSS declarations.
	// Appending ensures our rules take precedence.
	if err := htmlrewrite.AppendHeadContents(res, ruleInjection.Bytes()); err != nil {
		return fmt.Errorf("append head contents: %w", err)
	}

	return nil
}

func generateBatchedCSS(selectors []string) string {
	const batchSize = 100

	var builder strings.Builder
	for i := 0; i < len(selectors); i += batchSize {
		end := i + batchSize
		if end > len(selectors) {
			end = len(selectors)
		}
		batch := selectors[i:end]

		joinedSelectors := strings.Join(batch, ",")
		builder.WriteString(fmt.Sprintf("%s{display:none!important;}", joinedSelectors))
	}

	return builder.String()
}
