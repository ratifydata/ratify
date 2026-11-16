// Copyright (C) 2026 Ratify Data
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
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
	APIKey string `yaml:"API_KEY"`
	APIURL string `yaml:"API_URL"`
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "set user Api Key",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var setKey = &cobra.Command{
	Use:   "set-key <api-key>",
	Short: "Save the API key in .ratify/config.yml",
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
	if err != nil && !os.IsNotExist(err) {
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

	err = os.WriteFile(path, out, 0600)
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
	authCmd.AddCommand(setKey)
}
