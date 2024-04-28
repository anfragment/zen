package rule

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// TODO: generic mode
// see https://adguard.com/kb/general/ad-filtering/create-own-filters/#cookie-modifier

type cookieModifier struct {
	cookieName   string
	cookieRegexp *regexp.Regexp
}

var _ modifyingModifier = (*cookieModifier)(nil)

func (c cookieModifier) Parse(modifier string) error {
	modifierValue := strings.TrimPrefix(modifier, "cookie=")
	if modifierValue == "" || len(modifierValue) == len(modifier) {
		return errors.New("invalid cookie modifier")
	}

	regexp, err := parseRegexp(modifier)
	if err != nil {
		return fmt.Errorf("parse regexp: %w", err)
	}
	if regexp != nil {
		c.cookieRegexp = regexp
		return nil
	}

	c.cookieName = modifier
	return nil
}

func (c cookieModifier) ModifyReq(req *http.Request) (modified bool) {
	cookieHeader := req.Header.Get("Cookie")
	if cookieHeader == "" {
		return false
	}

	// See MDN's documentation on the Cookie header for more information:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cookie
	cookieParts := strings.Split(cookieHeader, ";")
	newCookieParts := make([]string, 0, len(cookieParts))
	for _, cookiePart := range cookieParts {
		if cookiePart == "" || cookiePart[0] != ' ' {
			// Non-standard cookie, but push it just in case it's valid
			// in the context of this particular request.
			newCookieParts = append(newCookieParts, cookiePart)
			continue
		}
		eqIndex := strings.IndexRune(cookiePart, '=')
		if eqIndex == -1 {
			// Non-standard cookie, but push it just in case it's valid
			// in the context of this particular request.
			newCookieParts = append(newCookieParts, cookiePart)
			continue
		}
		cookieName := cookiePart[1:eqIndex] // Start at 1 to skip leading space

		if c.cookieRegexp != nil && c.cookieRegexp.MatchString(cookieName) {
			continue
		}
		if c.cookieName == cookieName {
			continue
		}
		newCookieParts = append(newCookieParts, cookiePart)
	}

	if len(cookieParts) == len(newCookieParts) {
		return false
	}

	if len(newCookieParts) == 0 {
		req.Header.Del("Cookie")

	} else {
		req.Header.Set("Cookie", strings.Join(newCookieParts, ";"))
	}
	return true
}

func (c cookieModifier) ModifyRes(res *http.Response) (modified bool) {
	setCookieHeaders := res.Header.Values("Set-Cookie")
	if len(setCookieHeaders) == 0 {
		return false
	}

	// See MDN's documentation on the Set-Cookie header for more information:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie
	newSetCookieHeaders := make([]string, 0, len(setCookieHeaders))
	for _, setCookieHeader := range setCookieHeaders {
		eqIndex := strings.IndexRune(setCookieHeader, '=')
		if eqIndex == -1 {
			// Non-standard cookie, but push it just in case it's valid
			// in the context of this particular response.
			continue
		}
		cookieName := setCookieHeader[:eqIndex]
		if c.cookieRegexp != nil && c.cookieRegexp.MatchString(cookieName) {
			continue
		}
		if c.cookieName == cookieName {
			continue
		}
		newSetCookieHeaders = append(newSetCookieHeaders, setCookieHeader)
	}

	if len(setCookieHeaders) == len(newSetCookieHeaders) {
		return false
	}

	res.Header.Del("Set-Cookie")
	for _, newSetCookieHeader := range newSetCookieHeaders {
		res.Header.Add("Set-Cookie", newSetCookieHeader)
	}
	return true
}
