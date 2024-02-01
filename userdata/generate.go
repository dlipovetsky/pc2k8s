package userdata

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type Options struct {
	CAPXExecutableURL string

	Kubeconfig string

	SSHPublicKey string

	Endpoint              string
	Namespace             string
	Username              string
	Password              string
	AdditionalTrustBundle string
	Insecure              bool
	Categories            string
}

//go:embed user-data.yaml.gotemplate
var userDataGoTemplate string

var t = template.Must(template.New("userdata").Funcs(sprig.FuncMap()).Parse(userDataGoTemplate))

func String(opts Options) (string, error) {
	buf := bytes.Buffer{}
	err := t.Execute(&buf, opts)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
