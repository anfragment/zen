package matcher

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
)

// ruleModifiers represents modifiers of a rule.
type ruleModifiers struct {
	rule    string
	generic bool
	// basic modifiers
	// https://adguard.com/kb/general/ad-filtering/create-own-filters/#basic-rules-basic-modifiers
	domain domainModifiers
	// thirdParty optionType
	// header     string
	// important  optionType
	// method     string
	// content type modifiers
	// https://adguard.com/kb/general/ad-filtering/create-own-filters/#content-type-modifiers
	contentType bool
	document    bool
	font        bool
	image       bool
	media       bool
	script      bool
	stylesheet  bool
	other       bool
}

func (m *ruleModifiers) HandleRequest(req *http.Request) (*http.Request, *http.Response) {
	shouldBlock := false

	if len(m.domain) > 0 {
		referer := req.Header.Get("Referer")
		if url, err := url.Parse(referer); err == nil {
			hostname := url.Hostname()
			if !m.domain.MatchDomain(hostname) {
				shouldBlock = true
			}
		}
	}

	if m.contentType {
		modifiers := map[string]bool{
			"document": m.document,
			"font":     m.font,
			"image":    m.image,
			"audio":    m.media,
			"video":    m.media,
			"script":   m.script,
			"style":    m.stylesheet,
		}

		dest := req.Header.Get("Sec-Fetch-Dest")
		if val, ok := modifiers[dest]; ok {
			if !val {
				return req, nil
			}
		} else if !m.other {
			log.Printf("blocking with rule %s", m.rule)
			return req, nil
		}
	}
	if !m.generic {
		return req, nil
	}

	return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusForbidden, "blocked by zen")
}

func parseModifiers(modifiers string) (*ruleModifiers, error) {
	m := &ruleModifiers{}

	contentTypeInversed := false
	for _, modifier := range strings.Split(modifiers, ",") {
		if ind := strings.Index(modifier, "="); ind != -1 {
			switch modifier[:ind] {
			case "domain":
				domainModifiers, err := parseDomainModifiers(modifier[ind+1:])
				if err != nil {
					return nil, fmt.Errorf("invalid domain modifier %q: %w", modifier, err)
				}
				m.domain = domainModifiers
			default:
				return nil, fmt.Errorf("unknown modifier %q", modifier)
			}
			continue
		}

		if contentTypeInversed && modifier[0] != '~' {
			return nil, fmt.Errorf("mixing ~ and non-~ modifiers")
		}
		if modifier[0] == '~' {
			contentTypeInversed = true
			modifier = modifier[1:]
		}
		switch modifier {
		case "document":
			m.document = true
			m.contentType = true
		case "font":
			m.font = true
			m.contentType = true
		case "image":
			m.image = true
			m.contentType = true
		case "media":
			m.media = true
			m.contentType = true
		case "other":
			m.other = true
			m.contentType = true
		case "script":
			m.script = true
			m.contentType = true
		case "stylesheet":
			m.stylesheet = true
			m.contentType = true
		default:
			// first, do no harm
			// in case an unknown modifier is encountered, ignore the whole rule
			return nil, fmt.Errorf("unknown modifier %q", modifier)
		}
	}

	if contentTypeInversed {
		m.document = !m.document
		m.font = !m.font
		m.image = !m.image
		m.media = !m.media
		m.other = !m.other
		m.script = !m.script
		m.stylesheet = !m.stylesheet
	}

	return m, nil
}

type domainModifier struct {
	// https://adguard.com/kb/general/ad-filtering/create-own-filters/#domain-modifier
	invert  bool
	regular string
	tld     string
	regex   *regexp.Regexp
}

func (m *domainModifier) MatchDomain(domain string) bool {
	matches := false
	if m.regular != "" {
		matches = strings.HasSuffix(domain, m.regular)
	} else if m.tld != "" {
		matches = strings.HasPrefix(domain, m.tld)
	} else if m.regex != nil {
		matches = m.regex.MatchString(domain)
	}
	if m.invert {
		return !matches
	}
	return matches
}

type domainModifiers []domainModifier

func (m domainModifiers) MatchDomain(domain string) bool {
	for _, modifier := range m {
		if !modifier.MatchDomain(domain) {
			return false
		}
	}
	return true
}

func parseDomainModifiers(rule string) (domainModifiers, error) {
	modifiers := make([]domainModifier, 0, strings.Count(rule, "|")+1)
	for _, entry := range strings.Split(rule, "|") {
		if entry == "" {
			return nil, fmt.Errorf("empty modifier")
		}
		m := domainModifier{}
		if entry[0] == '~' {
			m.invert = true
			entry = entry[1:]
		}
		if entry[0] == '/' && entry[len(entry)-1] == '/' {
			regex, err := regexp.Compile(entry[1 : len(entry)-1])
			if err != nil {
				return nil, fmt.Errorf("invalid regex %q: %w", entry, err)
			}
			m.regex = regex
		} else if entry[len(entry)-1] == '*' {
			m.tld = entry[:len(entry)-1]
		} else {
			m.regular = entry
		}
	}
	return modifiers, nil
}
