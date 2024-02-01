package userdata

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestString(t *testing.T) {
	kubeconfig, err := os.ReadFile(filepath.Join("testdata", "kubeconfig"))
	if err != nil {
		t.Fatal("failed to read testdata kubeconfig")
	}

	type args struct {
		opts Options
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "simple",
			args: args{
				opts: Options{
					CAPXExecutableURL:     "https://example.com/foo",
					SSHPublicKey:          "foo",
					Kubeconfig:            string(kubeconfig),
					Endpoint:              "https://example.com:9440",
					Namespace:             "foo",
					Username:              "foo",
					Password:              "foo",
					AdditionalTrustBundle: "",
					Insecure:              true,
					Categories:            "",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userdata, err := String(tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("String() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			dummy := map[string]interface{}{}
			if err := yaml.UnmarshalStrict([]byte(userdata), dummy); err != nil {
				t.Errorf("Invalid YAML: %s", err)
			}
		})
	}
}