package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	qualitybash "github.com/ever-guild/quality-bash"
	"github.com/ever-guild/quality-bash/internal/install"
	"github.com/ever-guild/quality-bash/internal/quality"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var version = qualitybash.Version + "-dev"

type paths []string

func (p *paths) String() string     { return fmt.Sprint([]string(*p)) }
func (p *paths) Set(v string) error { *p = append(*p, v); return nil }
func run(ctx context.Context, args []string) (returnErr error) {
	if len(args) == 0 {
		return errors.New(usage)
	}
	if args[0] == "--help" || args[0] == "-h" {
		fmt.Println(usage)
		return nil
	}
	if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
		switch args[0] {
		case "version":
			fmt.Println("Usage: quality-bash version\nPrint the CLI version. No tools are required.")
			return nil
		case "tools":
			fmt.Println("Usage: quality-bash tools install [options]\nInstall the pinned tools explicitly. Use tools install --help for options.")
			return nil
		case "baseline":
			fmt.Println("Usage: quality-bash baseline check [options]\nRun fresh checks. Use baseline-update to save qualified results. Use the subcommand's --help for options.")
			return nil
		}
	}
	if args[0] == "version" {
		if len(args) != 1 {
			return fmt.Errorf("version accepts no arguments")
		}
		fmt.Println("quality-bash", version)
		return nil
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	command := args[0]
	if command == "tools" {
		if len(args) < 2 || args[1] != "install" {
			return fmt.Errorf("usage: tools install")
		}
		command = "tools install"
		args = append([]string{command}, args[2:]...)
	}
	toolsManifest := flags.String("tools-manifest", "", "pinned tool manifest (development packaging)")
	if command == "baseline" {
		if len(args) < 2 || args[1] != "check" {
			return fmt.Errorf("usage: baseline check [options]")
		}
		command += " " + args[1]
		args = append([]string{command}, args[2:]...)
	}
	if _, ok := commandHelp[command]; !ok {
		return fmt.Errorf("unknown command %q; use --help", command)
	}
	flags.SetOutput(os.Stdout)
	flags.Usage = func() {
		fmt.Printf("Usage: quality-bash %s [options]\n%s\n\nOptions:\n", command, commandHelp[command])
		flags.PrintDefaults()
	}
	baseline := flags.String("baseline", "quality-baseline.json", "baseline path relative to root")
	baseRevision := flags.String("base-revision", "", "also enforce baseline from this Git revision")
	create := flags.Bool("create", false, "explicitly create first baseline")
	migrate := flags.String("migrate", "", "explicit baseline migration explanation (baseline-update only)")
	root := flags.String("root", ".", "project root")
	reports := flags.String("reports", "quality-reports", "diagnostics directory relative to root")
	timeout := flags.Duration("timeout", 5*time.Minute, "timeout per external process")
	jobs := flags.Int("jobs", 1, "parallel mutation workers")
	var scripts, tests paths
	flags.Var(&scripts, "script", "selected Bash file (repeatable)")
	flags.Var(&tests, "test", "selected Bats file (repeatable)")
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if *jobs < 1 {
		return fmt.Errorf("jobs must be positive")
	}
	if (*create || *migrate != "") && command != "baseline-update" {
		return fmt.Errorf("--create and --migrate require baseline-update")
	}
	// Reject invalid policy documents before looking up or executing tools.
	if command == "qa" || command == "baseline check" || command == "baseline-update" {
		path := *baseline
		if !filepath.IsAbs(path) {
			path = filepath.Join(*root, path)
		}
		var policy quality.Baseline
		path, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if err := quality.ReadJSON(path, &policy); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		} else if err := policy.Check(policy); err != nil {
			return err
		}
		if *baseRevision != "" {
			if _, err := quality.ReadGitBaseline(ctx, *root, path, *baseRevision, *timeout); err != nil {
				return err
			}
		}
	}
	if err := quality.CheckBash(ctx, *timeout); err != nil {
		return err
	}
	if command == "tools install" || *toolsManifest != "" || len(install.BundledManifest) != 0 {
		data := install.BundledManifest
		if *toolsManifest != "" {
			var readErr error
			data, readErr = os.ReadFile(*toolsManifest)
			if readErr != nil {
				return readErr
			}
		}
		if len(data) == 0 {
			return fmt.Errorf("release tool manifest is not yet bundled; development builds require --tools-manifest")
		}
		manifest, manifestErr := install.ParseManifest(data)
		if manifestErr != nil {
			return manifestErr
		}
		manifest, manifestErr = install.PreferSystemKcov(ctx, manifest)
		if manifestErr != nil {
			return manifestErr
		}
		cache, cacheErr := install.CacheRoot()
		if cacheErr != nil {
			return cacheErr
		}
		var toolPaths []string
		var toolErr error
		if command == "tools install" {
			toolPaths, toolErr = install.InstallTools(ctx, cache, manifest)
		} else {
			toolPaths, toolErr = install.InstalledPaths(cache, manifest)
		}
		if toolErr != nil {
			return toolErr
		}
		if command == "tools install" {
			fmt.Println("Installed tools:", strings.Join(toolPaths, string(os.PathListSeparator)))
			return nil
		}
		if err := os.Setenv("PATH", strings.Join(toolPaths, string(os.PathListSeparator))+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
			return err
		}
	}
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if *jobs < 1 {
		return fmt.Errorf("jobs must be positive")
	}
	if args[0] == "doctor" {
		_, err := quality.Doctor(ctx, *timeout, os.Stdout)
		return err
	}
	absRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	reportPath := *reports
	if !filepath.IsAbs(reportPath) {
		reportPath = filepath.Join(absRoot, reportPath)
	}
	if command != "lint" && command != "test" && command != "mutation" && command != "coverage" && command != "qa" && command != "baseline check" && command != "baseline-update" {
		return fmt.Errorf("command %q is not yet implemented", args[0])
	}
	selection, err := quality.Discover(absRoot, scripts, tests)
	if err != nil {
		return err
	}
	runner := quality.Runner{Root: absRoot, Reports: reportPath, Timeout: *timeout, Stdout: os.Stdout, Stderr: os.Stderr}
	baselinePath := *baseline
	if !filepath.IsAbs(baselinePath) {
		baselinePath = filepath.Join(absRoot, baselinePath)
	}
	if err = quality.ValidateOutputPaths(absRoot, reportPath, baselinePath, selection); err != nil {
		return err
	}
	if command == "coverage" {
		report, err := runner.Coverage(ctx, selection)
		if err == nil {
			fmt.Printf("coverage: %d/%d\n", report.Score.Numerator, report.Score.Denominator)
		}
		return err
	}
	if command == "qa" || command == "baseline check" || command == "baseline-update" {
		var measurement quality.Measurement
		defer func() {
			if publishErr := quality.PublishGitHub(measurement, returnErr == nil); publishErr != nil && returnErr == nil {
				returnErr = publishErr
			}
		}()
		// Freeze policy before any project-controlled test can modify its files.
		var previous, target quality.Baseline
		var previousErr error
		if command != "baseline-update" {
			previousErr = quality.ReadJSON(baselinePath, &previous)
		}
		if *baseRevision != "" {
			target, err = quality.ReadGitBaseline(ctx, absRoot, baselinePath, *baseRevision, *timeout)
			if err != nil {
				return err
			}
		}
		if command == "qa" || command == "baseline check" {
			measurement, err = runner.Measure(ctx, selection, *jobs, baselinePath)
		} else {
			measurement, err = runner.LoadFresh(ctx, selection, *jobs, baselinePath)
		}
		if err != nil {
			return err
		}
		if command == "baseline-update" {
			if *migrate != "" {
				if *create || *baseRevision != "" {
					return fmt.Errorf("migration cannot be combined with --create or --base-revision; target comparisons remain incompatible until migration is accepted")
				}
				return quality.MigrateBaseline(baselinePath, measurement, measurement.Fingerprint, *migrate)
			}
			if *baseRevision != "" {
				if err = target.Check(measurement.Baseline); err != nil {
					return err
				}
			}
			return quality.UpdateBaseline(baselinePath, measurement, measurement.Fingerprint, *create)
		}
		if previousErr != nil {
			return previousErr
		}
		if err = previous.Check(measurement.Baseline); err != nil {
			return err
		}
		if *baseRevision != "" {
			if err = target.Check(previous); err != nil {
				return fmt.Errorf("working baseline regresses target revision: %w", err)
			}
			return target.Check(measurement.Baseline)
		}
		return nil
	}
	if args[0] == "lint" {
		return runner.Lint(ctx, selection)
	}
	if args[0] == "mutation" {
		report, err := runner.Mutate(ctx, selection, *jobs)
		if err == nil {
			fmt.Printf("mutation: %d/%d\n", report.Score.Numerator, report.Score.Denominator)
		}
		return err
	}
	return runner.Test(ctx, selection)
}
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "quality-bash:", err)
		os.Exit(1)
	}
}
