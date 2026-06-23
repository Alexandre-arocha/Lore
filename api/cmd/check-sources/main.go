// Command check-sources validates seed source configs against the GitHub API.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"lore/api/internal/sourcecheck"
)

func main() {
	seedPath := flag.String("seed", "seed/sources.json", "path to the sources seed JSON")
	only := flag.String("only", "", "comma-separated source slugs to check")
	timeout := flag.Duration("timeout", 2*time.Minute, "overall check timeout")
	repoMetadata := flag.Bool("repo-metadata", false, "also fetch repo metadata such as default_branch")
	flag.Parse()

	_ = godotenv.Load()

	sources, err := sourcecheck.LoadSeed(*seedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "check-sources: %v\n", err)
		os.Exit(1)
	}
	sources = filterSources(sources, *only)
	if len(sources) == 0 {
		fmt.Fprintln(os.Stderr, "check-sources: no sources selected")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	checker := sourcecheck.New(os.Getenv("GITHUB_TOKEN"), sourcecheck.WithRepoMetadata(*repoMetadata))
	var failed int
	for _, source := range sources {
		report, err := checker.Check(ctx, source)
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "FAIL %-16s %v\n", source.Slug, err)
			continue
		}

		branchNote := ""
		if report.DefaultBranch != "" && report.Branch != report.DefaultBranch {
			branchNote = " default=" + report.DefaultBranch
		}
		fmt.Printf(
			"OK   %-16s %-38s docs_path=%-34q files=%4d%s sample=%s\n",
			report.Slug,
			report.Repo+"@"+report.Branch,
			report.DocsPath,
			report.CandidateFiles,
			branchNote,
			strings.Join(report.Sample, ", "),
		)
	}

	if failed > 0 {
		fmt.Fprintf(os.Stderr, "check-sources: %d/%d sources failed\n", failed, len(sources))
		os.Exit(1)
	}
	fmt.Printf("check-sources: %d sources ok\n", len(sources))
}

func filterSources(sources []sourcecheck.Source, only string) []sourcecheck.Source {
	if strings.TrimSpace(only) == "" {
		return sources
	}

	selected := map[string]bool{}
	for _, slug := range strings.Split(only, ",") {
		if s := strings.TrimSpace(slug); s != "" {
			selected[s] = true
		}
	}

	filtered := make([]sourcecheck.Source, 0, len(selected))
	for _, source := range sources {
		if selected[source.Slug] {
			filtered = append(filtered, source)
		}
	}
	return filtered
}
