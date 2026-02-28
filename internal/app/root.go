package app

import (
	"context"
	"fmt"

	"github.com/alibilge/dirstral-cli/internal/breeze"
	"github.com/alibilge/dirstral-cli/internal/config"
	"github.com/alibilge/dirstral-cli/internal/host"
	"github.com/alibilge/dirstral-cli/internal/tempest"
	"github.com/spf13/cobra"
)

func Execute() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	root := newRootCommand(cfg)
	return root.Execute()
}

func newRootCommand(cfg config.Config) *cobra.Command {
	root := &cobra.Command{
		Use:   "dirstral",
		Short: "Terminal-first MCP coding companion",
		RunE: func(cmd *cobra.Command, args []string) error {
			for {
				choice, err := ShowStartScreen()
				if err != nil {
					return err
				}
				switch choice {
				case ChoiceBreeze:
					if err := runBreeze(cmd.Context(), cfg); err != nil {
						fmt.Println("error:", err)
					}
				case ChoiceTempest:
					opts := buildTempestOptions(cfg, cfg.MCP.URL, cfg.ElevenLabs.Voice, "", false, cfg.Verbose, cfg.ElevenLabs.BaseURL)
					if err := tempest.Run(cmd.Context(), opts); err != nil {
						fmt.Println("error:", err)
					}
				case ChoiceLighthouse:
					if err := runLighthouseMenu(cfg); err != nil {
						fmt.Println("error:", err)
					}
				default:
					fmt.Println("bye")
					return nil
				}
			}
		},
	}

	root.AddCommand(newBreezeCommand(cfg))
	root.AddCommand(newTempestCommand(cfg))
	root.AddCommand(newLighthouseCommand(cfg))
	return root
}

func newBreezeCommand(cfg config.Config) *cobra.Command {
	options := breeze.Options{MCPURL: cfg.MCP.URL, Transport: cfg.MCP.Transport, Model: cfg.Model, Verbose: cfg.Verbose}
	cmd := &cobra.Command{
		Use:   "breeze",
		Short: "Start text-to-text mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			return breeze.Run(cmd.Context(), options)
		},
	}
	cmd.Flags().StringVar(&options.MCPURL, "mcp", options.MCPURL, "MCP server URL")
	cmd.Flags().StringVar(&options.Transport, "transport", options.Transport, "MCP transport (streamable-http|stdio)")
	cmd.Flags().StringVar(&options.Model, "model", options.Model, "Model name")
	cmd.Flags().BoolVar(&options.Verbose, "verbose", options.Verbose, "Verbose MCP logging")
	return cmd
}

func newTempestCommand(cfg config.Config) *cobra.Command {
	var mcpURL string
	var voice string
	var device string
	var mute bool
	var verbose bool
	var baseURL string

	mcpURL = cfg.MCP.URL
	voice = cfg.ElevenLabs.Voice
	verbose = cfg.Verbose
	baseURL = cfg.ElevenLabs.BaseURL

	cmd := &cobra.Command{
		Use:   "tempest",
		Short: "Start voice mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := buildTempestOptions(cfg, mcpURL, voice, device, mute, verbose, baseURL)
			return tempest.Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&mcpURL, "mcp", mcpURL, "MCP server URL")
	cmd.Flags().StringVar(&voice, "voice", voice, "Voice id or name")
	cmd.Flags().StringVar(&device, "device", "", "Audio input device")
	cmd.Flags().BoolVar(&mute, "mute", false, "Disable TTS playback")
	cmd.Flags().BoolVar(&verbose, "verbose", verbose, "Verbose logging")
	cmd.Flags().StringVar(&baseURL, "elevenlabs-base-url", baseURL, "ElevenLabs base URL")
	return cmd
}

func newLighthouseCommand(cfg config.Config) *cobra.Command {
	var dir string
	var port int
	var listen string
	var mcpPath string
	var asJSON bool

	cmd := &cobra.Command{Use: "lighthouse", Short: "Host and monitor dir2mcp"}
	up := &cobra.Command{
		Use:   "up",
		Short: "Start dir2mcp and stream logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := host.UpOptions{
				Dir:     dir,
				Port:    port,
				Listen:  listen,
				MCPPath: mcpPath,
				JSON:    asJSON,
			}
			return host.Up(cmd.Context(), opts)
		},
	}
	up.Flags().StringVar(&dir, "dir", "", "Directory to serve/index")
	up.Flags().IntVar(&port, "port", 0, "Port for dir2mcp listen")
	up.Flags().StringVar(&listen, "listen", cfg.Host.Listen, "Listen host:port")
	up.Flags().StringVar(&mcpPath, "mcp-path", cfg.Host.MCPPath, "MCP endpoint path")
	up.Flags().BoolVar(&asJSON, "json", false, "Pass --json to dir2mcp up")

	status := &cobra.Command{Use: "status", Short: "Show host status", RunE: func(cmd *cobra.Command, args []string) error {
		return host.Status()
	}}
	down := &cobra.Command{Use: "down", Short: "Stop managed dir2mcp process", RunE: func(cmd *cobra.Command, args []string) error {
		return host.Down()
	}}

	cmd.AddCommand(up, status, down)
	return cmd
}

func runBreeze(ctx context.Context, cfg config.Config) error {
	return breeze.Run(ctx, breeze.Options{MCPURL: cfg.MCP.URL, Transport: cfg.MCP.Transport, Model: cfg.Model, Verbose: cfg.Verbose})
}

func buildTempestOptions(cfg config.Config, mcpURL, voice, device string, mute, verbose bool, baseURL string) tempest.Options {
	if mcpURL == "" {
		mcpURL = cfg.MCP.URL
	}
	if voice == "" {
		voice = cfg.ElevenLabs.Voice
	}
	if baseURL == "" {
		baseURL = cfg.ElevenLabs.BaseURL
	}
	return tempest.Options{
		MCPURL:  mcpURL,
		Voice:   voice,
		Device:  device,
		Mute:    mute,
		Verbose: verbose,
		BaseURL: baseURL,
	}
}

func runLighthouseMenu(cfg config.Config) error {
	for {
		choice, err := chooseLighthouseAction()
		if err != nil {
			return err
		}
		switch choice {
		case "Start Server":
			if err := host.Up(context.Background(), host.UpOptions{Listen: cfg.Host.Listen, MCPPath: cfg.Host.MCPPath}); err != nil {
				fmt.Println("error:", err)
			}
		case "Server Status":
			if err := host.Status(); err != nil {
				fmt.Println("error:", err)
			}
		case "Stop Server":
			if err := host.Down(); err != nil {
				fmt.Println("error:", err)
			}
		default:
			return nil
		}
	}
}
