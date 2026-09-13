package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/ratifydata/ratify/internal/schema"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type ConnectionDetails struct {
	Username string
	Password string
	Host     string
	Port     string
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	GroupID: "connect",
	Short: "Links to user's external database,list and verify connectivity",
	Long:  "Links to user's external database,list and verify connectivity",
	Run: func(cmd *cobra.Command, args []string) {

	}
}

var addConnectCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new external database to ratify",
	Long:  "Add a new external database to ratify",
	RunE: func(cmd *cobra.Command, args []string) error {
		connDetails,err := promptConnectionDetails()
		if err != nil{
			return err
		}
		schema.NewInspector(),
	}

}


func init() {
	connectCmd.AddCommand(addConnectCmd)
}

func promptConnectionDetails()  (*ConnectionDetails, error) {
	reader := bufio.NewReader(os.Stdin)

	hostname,err := prompt(reader, "Hostname: ")
	if err != nil {
		return nil, err
	}
	port,err := prompt(reader, "Port [5432]: ")
	if err != nil {
		return nil, err
	}

	username,err := prompt(reader, "Username: ")
	if err != nil {
		return nil, err
	}

	fmt.Println("Password: ")
	passBytes,err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, err
	}
	return &ConnectionDetails{
		Host: hostname,
		Username: username,
		Password: string(passBytes),
		Port: port,
	},nil
}

func prompt(reader *bufio.Reader,label string) (string, error)  {
	fmt.Printf("%s: ", label)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(value), nil
}


