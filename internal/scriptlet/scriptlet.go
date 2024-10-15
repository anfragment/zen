package scriptlet

import (
	"io"
	"text/template"
)

type scriptlet struct {
	Name string
	Args []string
}

var injectionTemplate = template.Must(template.New("scriptletInjection").Parse(`try {
  scriptlets.{{.Name}}({{range $index, $arg := .Args}}{{if $index}}, {{end}}{{printf "%q" $arg}}{{end}});
} catch (ex) {
  console.error(ex);
}
`))

func (r *scriptlet) GenerateInjection(w io.Writer) error {
	return injectionTemplate.Execute(w, r)
}
