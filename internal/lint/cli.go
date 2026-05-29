package lint

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/home-operations/yamlls/internal/config"
	"github.com/home-operations/yamlls/internal/diagnostics"
	"github.com/home-operations/yamlls/internal/schema"
	"github.com/home-operations/yamlls/internal/uri"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Run executes the `validate` subcommand. It resolves schemas exactly as
// the language server does, validates each path argument (directories are
// walked for *.yaml/*.yml), and prints diagnostics as
// `path:line:col: severity: message`. It returns the process exit code: 1
// if any error-severity diagnostic was reported, 2 on a usage or I/O error,
// 0 otherwise.
func Run(argv []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var root string
	flags.StringVar(&root, "root", "", "workspace root for .yamlls.yaml (default: auto-detect)")
	if err := flags.Parse(argv); err != nil {
		return 2
	}
	if flags.NArg() == 0 {
		fmt.Fprintln(stderr, "usage: yamlls validate [--root dir] <file|dir>...")
		return 2
	}

	files, err := collectYAML(flags.Args())
	if err != nil {
		fmt.Fprintf(stderr, "yamlls: %v\n", err)
		return 2
	}
	if len(files) == 0 {
		fmt.Fprintln(stderr, "yamlls: no YAML files found")
		return 2
	}

	if root == "" {
		root = findRoot(files[0])
	}
	ws, err := config.LoadFromWorkspace(uri.FromPath(root))
	if err != nil {
		fmt.Fprintf(stderr, "yamlls: %v\n", err)
		return 2
	}

	resolver := schema.NewResolver()
	resolver.SetSettings(ws)
	resolver.WaitForCatalog()
	store := schema.NewStore()

	failed := false
	for _, p := range files {
		b, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(stderr, "yamlls: %v\n", err)
			failed = true
			continue
		}
		text := string(b)
		diags := Document(text, p, resolver, store)
		diags = diagnostics.ParseSuppressions(text).Filter(diags)
		for _, d := range diags {
			fmt.Fprintln(stdout, formatDiagnostic(p, d))
			if severityOf(d) == protocol.DiagnosticSeverityError {
				failed = true
			}
		}
	}
	if failed {
		return 1
	}
	return 0
}

// collectYAML expands directory arguments into their *.yaml/*.yml files;
// explicit file arguments pass through regardless of extension.
func collectYAML(args []string) ([]string, error) {
	var out []string
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			out = append(out, arg)
			continue
		}
		err = filepath.WalkDir(arg, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && isYAML(path) {
				out = append(out, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func isYAML(p string) bool {
	ext := filepath.Ext(p)
	return ext == ".yaml" || ext == ".yml"
}

// findRoot walks up from the first file looking for .yamlls.yaml or a git
// repository, mirroring how an editor picks the workspace root. It falls
// back to the file's own directory.
func findRoot(file string) string {
	dir := filepath.Dir(file)
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	for {
		for _, marker := range []string{config.WorkspaceConfigFile, ".git"} {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Dir(file)
		}
		dir = parent
	}
}

func formatDiagnostic(path string, d protocol.Diagnostic) string {
	return fmt.Sprintf("%s:%d:%d: %s: %s",
		path, d.Range.Start.Line+1, d.Range.Start.Character+1,
		severityLabel(severityOf(d)), d.Message)
}

func severityOf(d protocol.Diagnostic) protocol.DiagnosticSeverity {
	if d.Severity != nil {
		return *d.Severity
	}
	return protocol.DiagnosticSeverityError
}

func severityLabel(s protocol.DiagnosticSeverity) string {
	switch s {
	case protocol.DiagnosticSeverityWarning:
		return "warning"
	case protocol.DiagnosticSeverityInformation:
		return "info"
	case protocol.DiagnosticSeverityHint:
		return "hint"
	default:
		return "error"
	}
}
