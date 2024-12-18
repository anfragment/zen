package cosmetic

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/anfragment/zen/internal/htmlrewrite"
	"github.com/anfragment/zen/internal/logger"
)

var (
	styleOpeningTag = []byte("<style>")
	styleClosingTag = []byte("</style>")
)

type Injector struct {
	// store stores and retrieves css by hostname.
	store Store
}

type Store interface {
	Add(hostnames []string, selector string)
	Get(hostname string) []string
}

func NewInjector(store Store) (*Injector, error) {
	if store == nil {
		return nil, errors.New("store is nil")
	}

	return &Injector{
		store: store,
	}, nil
}

func (inj *Injector) Inject(req *http.Request, res *http.Response) error {
	hostname := req.URL.Hostname()
	selectors := inj.store.Get(hostname)
	log.Printf("got %d cosmetic rules for %q", len(selectors), logger.Redacted(hostname))
	if len(selectors) == 0 {
		return nil
	}

	var ruleInjection bytes.Buffer
	ruleInjection.Write(styleOpeningTag)
	css := generateBatchedCSS(selectors)
	ruleInjection.WriteString(css)
	ruleInjection.Write(styleClosingTag)

	htmlrewrite.ReplaceHeadContents(res, func(match []byte) []byte {
		return bytes.Join([][]byte{match, ruleInjection.Bytes()}, nil)
	})

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

		joinedSelectors := strings.Join(batch, ", ")
		builder.WriteString(fmt.Sprintf("%s { display: none !important; }\n", joinedSelectors))
	}

	return builder.String()
}
