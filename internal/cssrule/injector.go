package cssrule

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
	RuleRegex          = regexp.MustCompile(`.*#@?\$#.+`)
	primaryRuleRegex   = regexp.MustCompile(`(.*?)#\$#(.*)`)
	exceptionRuleRegex = regexp.MustCompile(`(.*?)#@\$#(.+)`)

	injectionStart = []byte("<style>")
	injectionEnd   = []byte("</style>")
)

type store interface {
	AddPrimaryRule(hostnamePatterns string, css string) error
	AddExceptionRule(hostnamePatterns string, css string) error
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
	cssRules := inj.store.Get(hostname)
	log.Printf("got %d css rules for %q", len(cssRules), logger.Redacted(hostname))
	if len(cssRules) == 0 {
		return nil
	}

	var ruleInjection bytes.Buffer
	ruleInjection.Write(injectionStart)
	ruleInjection.WriteString(strings.Join(cssRules, ""))
	ruleInjection.Write(injectionEnd)

	htmlrewrite.ReplaceHeadContents(res, func(match []byte) []byte {
		return bytes.Join([][]byte{match, ruleInjection.Bytes()}, nil)
	})

	return nil
}
