package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	planemcp "github.com/dmipeck/plane-mcp/internal/mcp"
)

const (
	keyBaseURL = "base-url"
	keyAPIKey  = "api-key"
)

// Execute runs the root command.
func Execute() {
	if err := newRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "plane-mcp",
		Short:         "Multi-workspace Plane MCP server over stdio",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runRoot,
	}

	root.Flags().String("plane-base-url", "", "Plane API base URL (env PLANE_BASE_URL, default https://api.plane.so)")

	return root
}

func runRoot(cmd *cobra.Command, _ []string) error {
	conn, err := connectionFromCmd(cmd)
	if err != nil {
		return err
	}

	srv, err := planemcp.New(conn)
	if err != nil {
		return err
	}

	if err := srv.Run(cmd.Context()); err != nil {
		return fmt.Errorf("mcp stdio: %w", err)
	}
	return nil
}

// connectionFromCmd resolves Connection settings: flag > env > default for URL;
// API key from env only.
func connectionFromCmd(cmd *cobra.Command) (planemcp.Connection, error) {
	v := viper.New()
	v.SetEnvPrefix("PLANE")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.SetDefault(keyBaseURL, planemcp.DefaultBaseURL)

	if err := v.BindEnv(keyBaseURL); err != nil {
		return planemcp.Connection{}, err
	}
	if err := v.BindEnv(keyAPIKey); err != nil {
		return planemcp.Connection{}, err
	}
	if err := v.BindPFlag(keyBaseURL, cmd.Flags().Lookup("plane-base-url")); err != nil {
		return planemcp.Connection{}, err
	}

	return planemcp.Connection{
		BaseURL: v.GetString(keyBaseURL),
		APIKey:  v.GetString(keyAPIKey),
	}, nil
}
