package scriptlets

import "bytes"

func (i *Injector) CreateInjection(hostname string) ([]byte, error) {
	var buf bytes.Buffer

	for _, scriptlet := range i.universalScriptlets {
		if err := scriptlet.GenerateInjection(&buf); err != nil {
			return nil, err
		}
	}

	return nil, nil
}
