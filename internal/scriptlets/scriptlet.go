package scriptlets

import (
	"bytes"
	"text/template"
)

type Scriptlet struct {
	Name string
	Args []string
}

var injectionTemplate = template.Must(template.New("scriptletInjection").Parse(`try {
  scriptlets.{{.Name}}({{range $index, $arg := .Args}}{{if $index}}, {{end}}{{printf "%q" $arg}}{{end}});
} catch (ex) {
  console.error(ex);
}
`))

func (r *Scriptlet) GenerateInjection() []byte {
	var buf bytes.Buffer
	if err := injectionTemplate.Execute(&buf, r); err != nil {
		panic(err)
	}

	return buf.Bytes()
}
