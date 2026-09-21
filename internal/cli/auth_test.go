package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v3"
)

func writeCLIConfig(t *testing.T, content string) string {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.Mkdir(".ratify", 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(".ratify", "config.yml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSetAPIKey(t *testing.T) {
	for _, tt := range []struct {
		name, config, key string
		want              map[string]string
	}{
		{"add key", "HOST: localhost\n", "new-key", map[string]string{"HOST": "localhost", "API_KEY": "new-key"}},
		{"replace key", "API_KEY: old-key\nHOST: localhost\n", "new-key", map[string]string{"HOST": "localhost", "API_KEY": "new-key"}},
		{"empty config", "", "new-key", map[string]string{"API_KEY": "new-key"}},
		{"null config", "null\n", "new-key", map[string]string{"API_KEY": "new-key"}},
		{"special characters", "{}", "key: # 'quoted'\nnext", map[string]string{"API_KEY": "key: # 'quoted'\nnext"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := writeCLIConfig(t, tt.config)
			if err := setApiKey(tt.key); err != nil {
				t.Fatalf("setApiKey() error = %v", err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]string
			if err := yaml.Unmarshal(data, &got); err != nil {
				t.Fatalf("decode saved config: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("saved config = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetAPIKeyMissingConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := setApiKey("key"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("setApiKey() error = %v, want file not found", err)
	}
}

func TestSetAPIKeyInvalidConfig(t *testing.T) {
	for _, config := range []struct {
		param, setKey, name string
	}{{
		param:  "- one\n- two\n",
		setKey: "key",
		name:   "UnMarshalling Error",
	}, {
		param: "API_KEY",
		name:  "Marshalling Error",
	},
	} {
		t.Run(config.name, func(t *testing.T) {
			writeCLIConfig(t, config.param)
			if err := setApiKey(config.setKey); err == nil {
				t.Fatal("setApiKey() succeeded for invalid config")
			}
		})
	}
}

func TestSetAoiKeyMarshallingErr(t *testing.T) {

}
