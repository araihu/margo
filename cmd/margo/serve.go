package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/araihu/margo/internal/devserver"
	"github.com/araihu/margo/site"
	"github.com/cli/browser"
	"github.com/spf13/cobra"
)

type serveRequest struct {
	Input        string
	Host         string
	Port         int
	PortExplicit bool
	Open         bool
}

type serveFunc func(context.Context, serveRequest) error

type serveProject struct {
	deps       Dependencies
	root       string
	inputDir   string
	configPath string

	mu       sync.RWMutex
	output   string
	basePath string
}

func newServeCommand(deps Dependencies) *cobra.Command {
	host := "127.0.0.1"
	port := 0
	open := false
	diagnostics := string(diagnosticText)
	command := &cobra.Command{
		Use:   "serve [INPUT_DIR|CONFIG]",
		Short: "Serve a site with live reload for development",
		Long: "Build, watch, and serve a Margo site with live reload for development.\n\n" +
			"This development server is not for production use.",
		Args: func(command *cobra.Command, args []string) error {
			if len(args) <= 1 {
				return nil
			}
			return reportCommandError(command, diagnosticText, fmt.Errorf("cli.arguments_invalid: expected at most 1 input path, received %d", len(args)))
		},
		RunE: func(command *cobra.Command, args []string) error {
			input := "."
			if len(args) == 1 {
				input = args[0]
			}
			explicitPort := command.Flags().Changed("port")
			if explicitPort && (port < 1 || port > 65535) {
				return reportCommandError(command, diagnosticText, fmt.Errorf("serve.port_invalid: --port must be between 1 and 65535"))
			}
			if deps.ServeSite == nil {
				return reportCommandError(command, diagnosticText, fmt.Errorf("serve.unavailable: development server is unavailable"))
			}
			if err := deps.ServeSite(command.Context(), serveRequest{Input: input, Host: host, Port: port, PortExplicit: explicitPort, Open: open}); err != nil {
				return reportCommandError(command, diagnosticText, err)
			}
			return nil
		},
	}
	command.Flags().StringVar(&host, "host", host, "development server bind host")
	command.Flags().IntVar(&port, "port", port, "development server port; omitted selects an available port")
	command.Flags().BoolVar(&open, "open", open, "open the development site in the default browser")
	bindDiagnosticFlagErrors(command, &diagnostics)
	return command
}

func defaultServeFunc(deps Dependencies) serveFunc {
	return func(ctx context.Context, request serveRequest) error {
		return runDevelopmentServer(ctx, deps, request)
	}
}

func resolveServeProject(deps Dependencies, input string) (*serveProject, error) {
	if input == "" {
		input = "."
	}
	if !filepath.IsAbs(input) {
		input = filepath.Join(deps.WorkingDirectory, input)
	}
	absolute, err := filepath.Abs(input)
	if err != nil {
		return nil, fmt.Errorf("serve.input_invalid: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return nil, fmt.Errorf("serve.input_unreadable: %w", err)
	}
	project := &serveProject{deps: deps}
	if info.IsDir() {
		project.root = absolute
		candidate := filepath.Join(absolute, "site.yaml")
		if candidateInfo, candidateErr := os.Stat(candidate); candidateErr == nil && candidateInfo.Mode().IsRegular() {
			project.configPath = candidate
		} else {
			project.inputDir = absolute
		}
	} else {
		extension := strings.ToLower(filepath.Ext(absolute))
		if !info.Mode().IsRegular() || (extension != ".yaml" && extension != ".yml") {
			return nil, fmt.Errorf("serve.input_invalid: input file must be a .yaml or .yml site config")
		}
		project.root = filepath.Dir(absolute)
		project.configPath = absolute
	}
	if project.configPath != "" {
		if config, loadErr := site.LoadConfig(project.configPath); loadErr == nil {
			project.updateConfig(config, "")
		}
	}
	return project, nil
}

func (project *serveProject) Build(ctx context.Context) (devserver.Snapshot, error) {
	var result site.Result
	var err error
	if project.configPath != "" {
		config, loadErr := site.LoadConfig(project.configPath)
		if loadErr != nil {
			return devserver.Snapshot{}, loadErr
		}
		project.updateConfig(config, "")
		result, err = site.BuildConfig(ctx, site.ConfigRequest{
			ConfigPath:  project.configPath,
			Compiler:    compilerForPolicy(nil, policyTargetSite),
			AssetReader: project.deps.CheckAssetReader,
		})
		if err == nil {
			project.updateConfig(config, result.Site.BasePath)
		}
	} else {
		root, sources, discoverErr := discoverSiteSources(ctx, project.deps.SourceReader, project.inputDir)
		if discoverErr != nil {
			return devserver.Snapshot{}, discoverErr
		}
		result, err = site.Build(ctx, site.Request{
			SourceRoot:  root,
			Sources:     sources,
			Compiler:    compilerForPolicy(nil, policyTargetSite),
			Assets:      site.AssetsLocal,
			AssetReader: project.deps.CheckAssetReader,
		})
	}
	if err != nil {
		return devserver.Snapshot{}, err
	}
	manifest, err := marshalSiteManifest(result, "")
	if err != nil {
		return devserver.Snapshot{}, err
	}
	result.Artifacts = append(result.Artifacts, site.Artifact{Path: site.ManifestPath, Content: manifest})
	return devserver.NewSnapshot(result), nil
}

func (project *serveProject) updateConfig(config site.Config, resultBasePath string) {
	output := filepath.Join(filepath.Dir(project.configPath), filepath.FromSlash(config.Output))
	basePath := resultBasePath
	if basePath == "" {
		basePath = config.BasePath
	}
	project.mu.Lock()
	project.output = filepath.Clean(output)
	project.basePath = basePath
	project.mu.Unlock()
}

func (project *serveProject) Ignore(name string) bool {
	project.mu.RLock()
	output := project.output
	project.mu.RUnlock()
	return output != "" && pathWithin(output, name)
}

func (project *serveProject) BasePath() string {
	project.mu.RLock()
	defer project.mu.RUnlock()
	return project.basePath
}

func pathWithin(root, name string) bool {
	relative, err := filepath.Rel(root, name)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func runDevelopmentServer(ctx context.Context, deps Dependencies, request serveRequest) error {
	project, err := resolveServeProject(deps, request.Input)
	if err != nil {
		return err
	}
	listener, port, err := devserver.Listen(request.Host, request.Port, request.PortExplicit, nil)
	if err != nil {
		return err
	}
	changes, err := devserver.Watch(project.root, project.Ignore, 100*time.Millisecond)
	if err != nil {
		_ = listener.Close()
		return err
	}
	browserHost := request.Host
	if browserHost == "0.0.0.0" {
		browserHost = "127.0.0.1"
	} else if browserHost == "::" {
		browserHost = "::1"
	}
	url := devserver.URL(browserHost, port, project.BasePath())
	if !loopbackHost(request.Host) {
		fmt.Fprintln(deps.Stderr, "warning: development server has no authentication or TLS and is exposed beyond loopback")
	}
	return devserver.Run(ctx, devserver.Options{
		Listener: listener,
		URL:      url,
		Builder:  project,
		Changes:  changes,
		Started: func(url string) {
			fmt.Fprintln(deps.Stdout, "Margo development server (not for production)")
			fmt.Fprintf(deps.Stdout, "Serving %s\n", url)
			if request.Open {
				if openErr := browser.OpenURL(url); openErr != nil {
					fmt.Fprintf(deps.Stderr, "warning: could not open browser: %v\n", openErr)
				}
			}
		},
		BuildReported: func(event devserver.BuildEvent) {
			if event.Err != nil {
				_ = writeDiagnostic(deps.Stderr, diagnosticText, event.Err)
				return
			}
			verb := "rebuilt"
			if event.Initial {
				verb = "built"
			}
			fmt.Fprintf(deps.Stdout, "%s %d page(s), %d artifact(s); generation %d\n", verb, event.Snapshot.PageCount(), event.Snapshot.ArtifactCount(), event.Generation)
		},
	})
}

func loopbackHost(host string) bool {
	if strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}
