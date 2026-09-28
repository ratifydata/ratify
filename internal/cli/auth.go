package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

type Config struct {
	apiKey string `yaml:"API_KEY"`
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate user",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var setApiKeyCmd = &cobra.Command{
	Use:   "set-key <api-key>",
	Short: "Persists user API key in the environment",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := setApiKey(args[0])
		if err != nil {
			return errors.Wrap(err, "failed to set API key")
		}
		return nil
	},
}

func setApiKey(key string) error {
	var cfg map[string]string
	path := filepath.Join(".ratify", "config.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("Error reading config file")
		return err
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		fmt.Println("Error parsing config file")
		return err
	}
	if cfg == nil {
		cfg = make(map[string]string)
	}
	cfg["API_KEY"] = key
	out, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Println("Error marshalling config")
		return err
	}
	err = os.WriteFile(path, out, 0644)
	if err != nil {
		fmt.Println("Error writing config file")
	}
	return err
}

func init() {
	authCmd.AddGroup(&cobra.Group{
		ID:    "auth",
		Title: "Auth",
	})
	authCmd.AddCommand(setApiKeyCmd)
}
