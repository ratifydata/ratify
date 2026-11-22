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
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ratifydata/ratify/internal/contract"
	"github.com/stretchr/testify/require"
)

const contractTestID = "00000000-0000-0000-0000-000000000001"
const contractTeamID = "00000000-0000-0000-0000-000000000002"

func createContractArgs() []string {
	return []string{"create", "--connection-id", contractTestID, "--team-id", contractTeamID, "--name", "Orders", "--table", "orders"}
}

func TestContractCreateCommand(t *testing.T) {
	for _, sla := range []string{"omitted", "0", "12"} {
		t.Run(sla, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/api/cli/v1/contracts", r.URL.Path)
				require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				var params contract.ContractParams
				require.NoError(t, json.NewDecoder(r.Body).Decode(&params))
				require.Equal(t, contractTestID, params.ConnectionId.String())
				require.Equal(t, contractTeamID, params.TeamId.String())
				require.Equal(t, "Orders", params.DisplayName)
				require.Equal(t, "public", params.SchemaName)
				require.Equal(t, "orders", params.TableName)
				require.False(t, params.Description.Valid)
				if sla == "omitted" {
					require.Nil(t, params.FreshnessSlaHours)
				} else {
					require.NotNil(t, params.FreshnessSlaHours)
					if sla == "0" {
						require.Zero(t, *params.FreshnessSlaHours)
					} else {
						require.Equal(t, int32(12), *params.FreshnessSlaHours)
					}
				}
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"status":"ok","body":{"contract_id":"` + contractTestID + `","current_version":1,"status":"draft"}}`))
			}))
			defer server.Close()
			c := NewContractCmd(&Bridge{baseURL: server.URL, apiKey: "test-key", client: server.Client()})
			args := createContractArgs()
			if sla != "omitted" {
				args = append(args, "--freshness-sla-hours", sla)
			}
			out, err := executeCLI(t, c.Contract(), args...)
			require.NoError(t, err)
			require.Contains(t, out, "Created contract "+contractTestID+" (version 1, draft)")
		})
	}
}

func TestContractListCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/cli/v1/contracts", r.URL.Path)
		require.Equal(t, int64(0), r.ContentLength)
		require.Empty(t, r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"status":"ok","body":[{"id":"` + contractTestID + `","display_name":"Orders","schema_name":"public","table_name":"orders","status":"draft","current_version":1}]}`))
	}))
	defer server.Close()
	for _, output := range []string{"table", "json"} {
		t.Run(output, func(t *testing.T) {
			c := NewContractCmd(&Bridge{baseURL: server.URL, client: server.Client()})
			out, err := executeCLI(t, c.Contract(), "list", "--output", output)
			require.NoError(t, err)
			if output == "json" {
				var items []contract.ContractModel
				require.NoError(t, json.Unmarshal([]byte(out), &items))
				require.Len(t, items, 1)
				require.Equal(t, contractTestID, items[0].ID.String())
			} else {
				require.Contains(t, out, "public.orders")
				require.Contains(t, out, "Orders")
			}
		})
	}
}

func TestContractActivateCommand(t *testing.T) {
	for _, output := range []string{"text", "json"} {
		t.Run(output, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/api/cli/v1/contracts/"+contractTestID+"/activate", r.URL.Path)
				var params contract.ActivateContract
				require.NoError(t, json.NewDecoder(r.Body).Decode(&params))
				require.Equal(t, contractTestID, params.ID.String())
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			}))
			defer server.Close()
			c := NewContractCmd(&Bridge{baseURL: server.URL, client: server.Client()})
			out, err := executeCLI(t, c.Contract(), "activate", contractTestID, "--output", output)
			require.NoError(t, err)
			if output == "json" {
				require.JSONEq(t, `{"status":"ok"}`, out)
			} else {
				require.Contains(t, out, "Activated contract "+contractTestID)
			}
		})
	}
}

func TestContractCommandValidation(t *testing.T) {
	for _, args := range [][]string{
		{"create"}, {"create", "--connection-id", "bad"},
		append(createContractArgs(), "--team-id", "bad"),
		append(createContractArgs(), "--name", "  "),
		append(createContractArgs(), "--schema", "  "),
		append(createContractArgs(), "--table", "  "),
		append(createContractArgs(), "--freshness-sla-hours", "-1"),
		append(createContractArgs(), "--output", "xml"),
		{"activate"}, {"activate", "bad"}, {"activate", contractTestID, "extra"},
		{"activate", contractTestID, "--output", "xml"}, {"list", "extra"}, {"list", "--output", "xml"}, {"list", "--status", "draft"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			c := NewContractCmd(nil)
			_, err := executeCLI(t, c.Contract(), args...)
			require.Error(t, err)
		})
	}
}

func TestContractCommandsAPIErrors(t *testing.T) {
	for _, args := range [][]string{createContractArgs(), {"list"}, {"activate", contractTestID}} {
		t.Run(args[0], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"status":"error","message":"access denied"}`))
			}))
			defer server.Close()
			c := NewContractCmd(&Bridge{baseURL: server.URL, client: server.Client()})
			out, err := executeCLI(t, c.Contract(), args...)
			require.ErrorContains(t, err, "access denied")
			require.Empty(t, out)
		})
	}
}

func TestContractListEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","body":null}`))
	}))
	defer server.Close()
	for _, output := range []string{"table", "json"} {
		c := NewContractCmd(&Bridge{baseURL: server.URL, client: server.Client()})
		out, err := executeCLI(t, c.Contract(), "list", "--output", output)
		require.NoError(t, err)
		if output == "json" {
			require.JSONEq(t, `[]`, out)
		} else {
			require.Equal(t, "No contracts.\n", out)
		}
	}
}

func TestContractRootRegistration(t *testing.T) {
	cmd, _, err := newRootCmd().Find([]string{"contract", "create"})
	require.NoError(t, err)
	require.Equal(t, "create", cmd.Name())
}

func TestInteractiveContractCreate(t *testing.T) {
	for _, sla := range []string{"", "0", "12"} {
		t.Run("sla="+sla, func(t *testing.T) {
			created := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasSuffix(r.URL.Path, "/connections"):
					_, _ = w.Write([]byte(`{"status":"ok","body":[{"id":"` + contractTestID + `","display_name":"Warehouse"}]}`))
				case strings.HasSuffix(r.URL.Path, "/teams"):
					_, _ = w.Write([]byte(`{"status":"ok","body":[{"id":"` + contractTeamID + `","name":"Data Platform"}]}`))
				default:
					require.Equal(t, http.MethodPost, r.Method)
					var params contract.ContractParams
					require.NoError(t, json.NewDecoder(r.Body).Decode(&params))
					require.Equal(t, contractTestID, params.ConnectionId.String())
					require.Equal(t, "public", params.SchemaName)
					require.Equal(t, "orders", params.TableName)
					require.Equal(t, contractTeamID, params.TeamId.String())
					require.Equal(t, "Orders", params.DisplayName)
					require.Equal(t, "Order events", params.Description.String)
					require.True(t, params.Description.Valid)
					if sla == "" {
						require.Nil(t, params.FreshnessSlaHours)
					} else {
						require.NotNil(t, params.FreshnessSlaHours)
					}
					created = true
					_, _ = w.Write([]byte(`{"status":"ok","body":{"contract_id":"` + contractTestID + `","current_version":1,"status":"draft"}}`))
				}
			}))
			defer server.Close()
			c := NewContractCmd(&Bridge{baseURL: server.URL, client: server.Client()})
			cmd := c.Contract()
			var out, prompts bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&prompts)
			cmd.SetIn(strings.NewReader("invalid\n1\n1\n\norders\n\nOrders\nOrder events\n" + sla + "\n"))
			cmd.SetArgs([]string{"create", "--output", "json"})
			require.NoError(t, cmd.Execute())
			require.True(t, created)
			require.Contains(t, prompts.String(), "Warehouse")
			require.Contains(t, prompts.String(), "Data Platform")
			require.True(t, json.Valid(out.Bytes()))
		})
	}
}

func TestInteractiveContractCreateEOF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{"status":"ok","body":[{"id":"` + contractTestID + `"}]}`))
	}))
	defer server.Close()
	c := NewContractCmd(&Bridge{baseURL: server.URL, client: server.Client()})
	cmd := c.Contract()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"create"})
	require.ErrorContains(t, cmd.Execute(), "EOF")
}

func TestContractShow(t *testing.T) {
	body := `{"id":"` + contractTestID + `","display_name":"Orders","schema_name":"sales","table_name":"orders","status":"active","current_version":2,"freshness_sla_hours":0,"producer":{"id":"` + contractTeamID + `","name":"Data Platform","email":"data@example.com"},"consumers":[{"team":{"id":"` + contractTeamID + `","name":"Analytics"},"status":"approved","usage_description":"Reporting"}],"versions":[{"version":3,"columns":[{"column_name":"future_column"}]},{"version":2,"description":"Added order ID","columns":[{"column_name":"order_id","data_type":"integer","is_primary_key":true,"position":1}]},{"version":1,"columns":[]}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/cli/v1/contracts/"+contractTestID, r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"status":"ok","body":` + body + `}`))
	}))
	defer server.Close()
	for _, format := range []string{"text", "json"} {
		c := NewContractCmd(&Bridge{baseURL: server.URL, apiKey: "test-key", client: server.Client()})
		out, err := executeCLI(t, c.Contract(), "show", contractTestID, "--output", format)
		require.NoError(t, err)
		if format == "json" {
			var detail contract.ContractModel
			require.NoError(t, json.Unmarshal([]byte(out), &detail))
			require.Len(t, detail.Versions, 3)
		} else {
			for _, text := range []string{"Orders", "sales.orders", "order_id", "Added order ID", "0 hours", "Version history"} {
				require.Contains(t, out, text)
			}
			require.NotContains(t, out, "future_column")
		}
	}
}

func TestContractShowEmptyDetailsAndErrors(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{`{"status":"ok","body":{"current_version":1}}`, ""},
		{`{"status":"ok","body":null}`, "no details"},
		{`{"status":"error","message":"contract not found"}`, "contract not found"},
	} {
		t.Run(tc.body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer server.Close()
			c := NewContractCmd(&Bridge{baseURL: server.URL, client: server.Client()})
			out, err := executeCLI(t, c.Contract(), "show", contractTestID)
			if tc.want != "" {
				require.ErrorContains(t, err, tc.want)
				require.Empty(t, out)
			} else {
				require.NoError(t, err)
				for _, text := range []string{"No versions.", "No columns recorded", "Not set"} {
					require.Contains(t, out, text)
				}
			}
		})
	}
}

func TestContractShowValidation(t *testing.T) {
	for _, args := range [][]string{{"show"}, {"show", "bad"}, {"show", contractTestID, "extra"}, {"show", contractTestID, "--output", "xml"}} {
		c := NewContractCmd(nil)
		_, err := executeCLI(t, c.Contract(), args...)
		require.Error(t, err)
	}
	cmd, _, err := newRootCmd().Find([]string{"contract", "show"})
	require.NoError(t, err)
	require.Equal(t, "show", cmd.Name())
}

func TestContractCreateRejectsInvalidFlagsBeforeRequests(t *testing.T) {
	for _, flags := range [][]string{
		{"--team-id", "bad"}, {"--connection-id", "bad"},
		{"--name", " "}, {"--table", ""}, {"--schema", " "},
		{"--connection-id", ""}, {"--team-id", ""}, {"--freshness-sla-hours", "-1"},
	} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected request") }))
			defer server.Close()
			c := NewContractCmd(&Bridge{baseURL: server.URL, client: server.Client()})
			cmd := c.Contract()
			var prompts bytes.Buffer
			cmd.SetErr(&prompts)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetIn(strings.NewReader(""))
			cmd.SetArgs(append([]string{"create"}, flags...))
			require.Error(t, cmd.Execute())
			require.NotContains(t, prompts.String(), "Choice:")
		})
	}
}

func TestContractCreateSelectionErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		team             bool
	}{
		{"no connections", `{"status":"ok","body":[]}`, "no connection available", false},
		{"null connections", `{"status":"ok","body":null}`, "no connection available", false},
		{"no teams", `{"status":"ok","body":[]}`, "no producer team available", true},
		{"denied", `{"status":"error","message":"access denied"}`, "access denied", false},
		{"invalid JSON", `{`, "decode API response", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			c := NewContractCmd(&Bridge{baseURL: server.URL, client: server.Client()})
			cmd := c.Contract()
			cmd.SetIn(strings.NewReader(""))
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			args := []string{"create"}
			if tc.team {
				args = append(args, "--connection-id", contractTestID)
			}
			cmd.SetArgs(args)
			require.ErrorContains(t, cmd.Execute(), tc.want)
		})
	}
}

func TestContractCreateInteractiveFlagsAndFreshnessRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/cli/v1/contracts", r.URL.Path)
		var params contract.ContractParams
		require.NoError(t, json.NewDecoder(r.Body).Decode(&params))
		require.Equal(t, "Orders", params.DisplayName)
		require.Equal(t, "From flag", params.Description.String)
		require.NotNil(t, params.FreshnessSlaHours)
		require.Equal(t, int32(12), *params.FreshnessSlaHours)
		_, _ = w.Write([]byte(`{"status":"ok","body":{"contract_id":"` + contractTestID + `","current_version":1,"status":"draft"}}`))
	}))
	defer server.Close()
	c := NewContractCmd(&Bridge{baseURL: server.URL, client: server.Client()})
	cmd := c.Contract()
	var prompts bytes.Buffer
	cmd.SetErr(&prompts)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("-1\nabc\n2147483648\n12"))
	cmd.SetArgs(append(createContractArgs(), "--schema", "public", "--description", "From flag", "--interactive"))
	require.NoError(t, cmd.Execute())
	require.Equal(t, 3, strings.Count(prompts.String(), "Enter a nonnegative"))
	require.NotContains(t, prompts.String(), "Display name:")
}

func TestContractCreateMissingResponseDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","body":null}`))
	}))
	defer server.Close()
	c := NewContractCmd(&Bridge{baseURL: server.URL, client: server.Client()})
	out, err := executeCLI(t, c.Contract(), createContractArgs()...)
	require.ErrorContains(t, err, "no valid contract details")
	require.Empty(t, out)
}

func TestContractSelectionRetriesAndUsesSelectedIndex(t *testing.T) {
	var out bytes.Buffer
	choice, err := selectContractItem(bufio.NewReader(strings.NewReader("0\n-1\n3\nabc\n2\n")), &out, "connection", []string{"First", "Second"})
	require.NoError(t, err)
	require.Equal(t, 1, choice)
	require.Equal(t, 4, strings.Count(out.String(), "Enter a number"))
}

func TestContractCreateNilBridge(t *testing.T) {
	for _, args := range [][]string{{"create"}, createContractArgs()} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			c := NewContractCmd(nil)
			out, err := executeCLI(t, c.Contract(), args...)
			require.ErrorContains(t, err, "API connection is required")
			require.Empty(t, out)
		})
	}
}

func TestContractDeprecateConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name, input   string
		requests      int
		want, wantErr string
	}{
		{"confirm", "y\n", 1, "Deprecated contract " + contractTestID, ""},
		{"uppercase", "Y\n", 1, "Deprecated contract", ""},
		{"decline", "n\n", 0, "Deprecation canceled.", ""},
		{"uppercase decline", "N\n", 0, "Deprecation canceled.", ""},
		{"retry", "\nyes\ninvalid\ny\n", 1, "Deprecated contract", ""},
		{"EOF", "", 0, "", "EOF"},
		{"invalid then EOF", "yes\n", 0, "", "EOF"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var prompts bytes.Buffer
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Contains(t, prompts.String(), "[y/n]")
				require.Equal(t, http.MethodPatch, r.Method)
				require.Equal(t, "/api/cli/v1/contracts/"+contractTestID+"/deprecate", r.URL.Path)
				require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				require.Zero(t, r.ContentLength)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			}))
			defer server.Close()
			c := NewContractCmd(&Bridge{baseURL: server.URL, apiKey: "test-key", client: server.Client()})
			cmd := c.Contract()
			var out bytes.Buffer
			cmd.SetIn(strings.NewReader(tc.input))
			cmd.SetOut(&out)
			cmd.SetErr(&prompts)
			cmd.SetArgs([]string{"deprecate", contractTestID})
			err := cmd.Execute()
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
				require.Contains(t, out.String(), tc.want)
			}
			require.Equal(t, tc.requests, calls)
		})
	}
}

func TestContractDeprecateValidation(t *testing.T) {
	for _, args := range [][]string{{"deprecate"}, {"deprecate", "bad"}, {"deprecate", contractTestID, "extra"}, {"deprecate", contractTestID}} {
		c := NewContractCmd(nil)
		_, err := executeCLI(t, c.Contract(), args...)
		require.Error(t, err)
	}
	cmd, _, err := newRootCmd().Find([]string{"contract", "deprecate"})
	require.NoError(t, err)
	require.Equal(t, "deprecate", cmd.Name())
}

func TestContractDeprecateAPIErrors(t *testing.T) {
	for _, body := range []string{`{"status":"error","message":"contract not found"}`, `{"status":"error","message":"only active contracts can be deprecated"}`, `{`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			c := NewContractCmd(&Bridge{baseURL: server.URL, client: server.Client()})
			cmd := c.Contract()
			var out bytes.Buffer
			cmd.SetIn(strings.NewReader("y\n"))
			cmd.SetOut(&out)
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs([]string{"deprecate", contractTestID})
			require.Error(t, cmd.Execute())
			require.NotContains(t, out.String(), "Deprecated contract")
		})
	}
}
