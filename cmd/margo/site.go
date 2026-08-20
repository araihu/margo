package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/site"
	"github.com/spf13/cobra"
)

const (
	siteReportSchema   = "margo-site-report/v1"
	siteManifestSchema = "margo-site-manifest/v1"
)

type siteReport struct {
	SchemaVersion string      `json:"schemaVersion"`
	Artifacts     int         `json:"artifacts"`
	Manifest      string      `json:"manifest"`
	Policy        string      `json:"policy,omitempty"`
	Pages         []site.Page `json:"pages"`
}

type siteManifestDocument struct {
	SchemaVersion       string              `json:"schemaVersion"`
	Digest              string              `json:"digest"`
	Policy              string              `json:"policy,omitempty"`
	ConfigVersion       int                 `json:"configVersion,omitempty"`
	Layout              string              `json:"layout,omitempty"`
	LayoutSchemaHash    string              `json:"layoutSchemaHash,omitempty"`
	BaseURL             string              `json:"baseURL,omitempty"`
	BasePath            string              `json:"basePath,omitempty"`
	DocumentStyleDigest string              `json:"documentStyleDigest,omitempty"`
	Routes              []site.Page         `json:"routes,omitempty"`
	Entries             []siteManifestEntry `json:"entries"`
}

type siteManifestEntry struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

func newSiteCommand(deps Dependencies) *cobra.Command {
	var outputDirectory string
	assets := string(site.AssetsLocal)
	diagnostics := string(diagnosticText)
	var policyOptions policyFlags
	command := &cobra.Command{
		Use:   "site INPUT_DIR|CONFIG",
		Short: "Build a linked HTML site from a Markdown directory",
		Args:  diagnosticExactArgs(1, &diagnostics),
		RunE: func(command *cobra.Command, args []string) error {
			format, err := parseDiagnosticFormat(diagnostics)
			if err != nil {
				return err
			}
			configInput := false
			if info, statErr := os.Stat(args[0]); statErr == nil {
				configInput = !info.IsDir() && (strings.HasSuffix(strings.ToLower(args[0]), ".yaml") || strings.HasSuffix(strings.ToLower(args[0]), ".yml"))
			}
			if outputDirectory == "" && !configInput {
				return reportCommandError(command, format, errors.New("site.output_required: --output-dir is required"))
			}
			mode := site.AssetMode(assets)
			if mode != site.AssetsLocal && mode != site.AssetsInline {
				return reportCommandError(command, format, errors.New("site.assets_invalid: --assets must be local or inline"))
			}
			policy, err := policyOptions.load(command.Context(), deps.SourceReader)
			if err != nil {
				return reportCommandError(command, format, err)
			}
			var result site.Result
			if configInput {
				config, configErr := site.LoadConfig(args[0])
				if configErr != nil {
					return reportCommandError(command, format, configErr)
				}
				if outputDirectory == "" {
					outputDirectory = filepath.Join(filepath.Dir(args[0]), config.Output)
				}
				result, err = site.BuildConfig(command.Context(), site.ConfigRequest{ConfigPath: args[0], Compiler: compilerForPolicy(policy, policyTargetSite), AssetReader: deps.CheckAssetReader})
			} else {
				if _, err := validateSiteOutputTarget(outputDirectory); err != nil {
					return reportCommandError(command, format, err)
				}
				root, sources, discoverErr := discoverSiteSources(command.Context(), deps.SourceReader, args[0])
				if discoverErr != nil {
					return reportCommandError(command, format, discoverErr)
				}
				result, err = site.Build(command.Context(), site.Request{
					SourceRoot: root, Sources: sources, Compiler: compilerForPolicy(policy, policyTargetSite),
					Assets: mode, AssetReader: deps.CheckAssetReader,
				})
			}
			if err != nil {
				return reportCommandError(command, format, err)
			}
			if _, err := validateSiteOutputTarget(outputDirectory); err != nil {
				return reportCommandError(command, format, err)
			}
			policyDigest := ""
			if policy != nil {
				policyDigest = policy.Digest
			}
			if err := publishSite(command.Context(), outputDirectory, result, policyDigest); err != nil {
				return reportCommandError(command, format, err)
			}
			return writeSiteReport(command.OutOrStdout(), format, siteReport{
				SchemaVersion: siteReportSchema, Artifacts: len(result.Artifacts), Manifest: result.Manifest.Digest(), Policy: policyDigest, Pages: result.Pages,
			})
		},
	}
	command.Flags().StringVar(&outputDirectory, "output-dir", "", "new directory for generated site files")
	command.Flags().StringVar(&assets, "assets", string(site.AssetsLocal), "asset mode: local or inline")
	command.Flags().StringVar(&diagnostics, "diagnostics", string(diagnosticText), "diagnostic format: text or json")
	policyOptions.bind(command)
	bindDiagnosticFlagErrors(command, &diagnostics)
	return command
}

func discoverSiteSources(ctx context.Context, reader SourceReader, input string) (string, []site.Source, error) {
	if input == "" || input == "-" {
		return "", nil, errors.New("site.input_invalid: INPUT_DIR must be a directory path")
	}
	absolute, err := filepath.Abs(input)
	if err != nil {
		return "", nil, fmt.Errorf("site.input_invalid: %w", err)
	}
	realRoot, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", nil, fmt.Errorf("site.input_unreadable: %w", err)
	}
	info, err := os.Stat(realRoot)
	if err != nil {
		return "", nil, fmt.Errorf("site.input_unreadable: %w", err)
	}
	if !info.IsDir() {
		return "", nil, errors.New("site.input_invalid: INPUT_DIR is not a directory")
	}
	if reader == nil {
		return "", nil, errors.New("site.input_reader_required: file reader is unavailable")
	}
	paths := make([]string, 0)
	err = filepath.WalkDir(realRoot, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		entryInfo, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if !entryInfo.Mode().IsRegular() {
			return fmt.Errorf("site.input_invalid: %s is not a regular file", name)
		}
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if extension == ".md" || extension == ".markdown" {
			paths = append(paths, name)
		}
		return nil
	})
	if err != nil {
		return "", nil, fmt.Errorf("site.input_unreadable: %w", err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return "", nil, errors.New("site.sources_empty: INPUT_DIR contains no Markdown files")
	}
	sources := make([]site.Source, 0, len(paths))
	for _, name := range paths {
		if err := ctx.Err(); err != nil {
			return "", nil, err
		}
		content, err := reader.ReadFile(name, margo.MaxDocumentBytes)
		if err != nil {
			return "", nil, fmt.Errorf("site.input_read: %s: %w", name, err)
		}
		if int64(len(content)) > margo.MaxDocumentBytes {
			return "", nil, fmt.Errorf("site.input_too_large: %s exceeds %d bytes", name, margo.MaxDocumentBytes)
		}
		relative, err := filepath.Rel(realRoot, name)
		if err != nil {
			return "", nil, fmt.Errorf("site.input_invalid: %w", err)
		}
		sources = append(sources, site.Source{Path: filepath.ToSlash(relative), Content: append([]byte(nil), content...)})
	}
	return realRoot, sources, nil
}

func publishSite(ctx context.Context, target string, result site.Result, policy string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	absolute, err := validateSiteOutputTarget(target)
	if err != nil {
		return err
	}
	parent := filepath.Dir(absolute)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("site.output_create: %w", err)
	}
	staging, err := os.MkdirTemp(parent, ".margo-site-")
	if err != nil {
		return fmt.Errorf("site.output_create: %w", err)
	}
	defer os.RemoveAll(staging)
	for _, artifact := range result.Artifacts {
		if err := writeSiteArtifact(ctx, staging, artifact.Path, artifact.Content); err != nil {
			return err
		}
	}
	manifest, err := marshalSiteManifest(result, policy)
	if err != nil {
		return err
	}
	if err := writeSiteArtifact(ctx, staging, site.ManifestPath, manifest); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := renameSiteDirectoryNoReplace(staging, absolute); err != nil {
		if _, statErr := os.Lstat(absolute); statErr == nil {
			return fmt.Errorf("site.output_exists: %s already exists", absolute)
		}
		return fmt.Errorf("site.output_commit: %w", err)
	}
	if err := syncSiteOutputParent(parent); err != nil {
		return fmt.Errorf("site.output_durability: output is visible but parent synchronization failed: %w", err)
	}
	return nil
}

func validateSiteOutputTarget(target string) (string, error) {
	absolute, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("site.output_invalid: %w", err)
	}
	if _, err := os.Lstat(absolute); err == nil {
		return "", fmt.Errorf("site.output_exists: %s already exists; choose a new --output-dir", absolute)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("site.output_invalid: %w", err)
	}
	return absolute, nil
}

func writeSiteArtifact(ctx context.Context, root, relative string, content []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	within, err := filepath.Rel(root, target)
	if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return fmt.Errorf("site.artifact_invalid: %q escapes the output directory", relative)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("site.output_create: %w", err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		return fmt.Errorf("site.output_write: %w", err)
	}
	return nil
}

func marshalSiteManifest(result site.Result, policy string) ([]byte, error) {
	if err := result.Manifest.Validate(); err != nil {
		return nil, fmt.Errorf("site.manifest_invalid: %w", err)
	}
	document := siteManifestDocument{SchemaVersion: siteManifestSchema, Digest: result.Manifest.Digest(), Policy: policy, ConfigVersion: result.Site.ConfigVersion, Layout: result.Site.Layout, LayoutSchemaHash: result.Site.LayoutSchemaHash, BaseURL: result.Site.BaseURL, BasePath: result.Site.BasePath, DocumentStyleDigest: result.Site.DocumentStyleDigest, Routes: append([]site.Page(nil), result.Site.Routes...), Entries: make([]siteManifestEntry, len(result.Manifest.Entries))}
	for index, entry := range result.Manifest.Entries {
		document.Entries[index] = siteManifestEntry{Path: entry.Path, Digest: entry.Digest.String()}
	}
	data, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("site.manifest_invalid: %w", err)
	}
	return append(data, '\n'), nil
}

func writeSiteReport(writer io.Writer, format diagnosticFormat, report siteReport) error {
	if writer == nil {
		return errors.New("cli.diagnostics_writer_required: output writer is unavailable")
	}
	if format == diagnosticJSON {
		return json.NewEncoder(writer).Encode(report)
	}
	if format != diagnosticText {
		return errors.New("cli.diagnostics_invalid: diagnostics must be text or json")
	}
	for _, page := range report.Pages {
		if _, err := fmt.Fprintf(writer, "%s -> %s\n", page.Source, page.Output); err != nil {
			return err
		}
	}
	policy := ""
	if report.Policy != "" {
		policy = "; policy " + report.Policy
	}
	_, err := fmt.Fprintf(writer, "built %d page(s), %d artifact(s); manifest %s%s\n", len(report.Pages), report.Artifacts, report.Manifest, policy)
	return err
}
