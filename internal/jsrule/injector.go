package jsrule

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/anfragment/zen/internal/hostmatch"
	"github.com/anfragment/zen/internal/htmlrewrite"
)

type store interface {
	AddPrimaryRule(hostnamePatterns string, script string) error
	AddExceptionRule(hostnamePatterns string, script string) error
	Get(hostname string) []string
}

type Injector struct {
	store store
}

var (
	RuleRegex          = regexp.MustCompile(`.*#@?%#.+`)
	primaryRuleRegex   = regexp.MustCompile(`(.*)#%#(.+)`)
	exceptionRuleRegex = regexp.MustCompile(`(.*)#@%#(.+)`)

	scriptOpeningTag = []byte("<script>")
	scriptClosingTag = []byte("</script>")
)

func NewInjector() *Injector {
	return &Injector{
		store: hostmatch.NewHostMatcher[string](),
	}
}

func (i *Injector) AddRule(rule string) error {
	if match := primaryRuleRegex.FindStringSubmatch(rule); match != nil {
		if err := i.store.AddPrimaryRule(match[1], match[2]); err != nil {
			return fmt.Errorf("add primary rule: %w", err)
		}
		return nil
	}

	if match := exceptionRuleRegex.FindStringSubmatch(rule); match != nil {
		if err := i.store.AddExceptionRule(match[1], match[2]); err != nil {
			return fmt.Errorf("add exception rule: %w", err)
		}
		return nil
	}

	return errors.New("unsupported syntax")
}

func (inj *Injector) Inject(req *http.Request, res *http.Response) error {
	hostname := req.URL.Hostname()
	scripts := inj.store.Get(hostname)
	log.Printf("got %d scripts for %q", len(scripts), hostname)
	if len(scripts) == 0 {
		return nil
	}

	var injection []byte
	injection = append(injection, scriptOpeningTag...)
	for _, script := range scripts {
		injection = append(injection, script...)
		if len(script) > 0 && script[len(script)-1] != ';' {
			injection = append(injection, ';')
		}
	}
	injection = append(injection, scriptClosingTag...)

	htmlrewrite.ReplaceBodyContents(res, func(match []byte) []byte {
		match = append(match, injection...)
		return match
	})

	return nil
}
