package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhonsferg/traverse"
)

func main() {
	var (
		metadataURL = flag.String("url", "", "OData metadata URL (e.g., https://api.example.com/odata/v4/$metadata)")
		pkgName     = flag.String("package", "", "Output package name (e.g., 'generated')")
		outputDir   = flag.String("output", "", "Output directory for generated files")
		help        = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help || *metadataURL == "" || *pkgName == "" || *outputDir == "" {
		printUsage()
		os.Exit(1)
	}

	if err := run(*metadataURL, *pkgName, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully generated OData client in %s\n", *outputDir)
}

func run(metadataURL, pkgName, outputDir string) error {
	// Fetch metadata from the URL
	fmt.Printf("Fetching metadata from %s...\n", metadataURL)
	resp, err := http.Get(metadataURL)
	if err != nil {
		return fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("metadata request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse metadata
	fmt.Println("Parsing metadata...")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read metadata body: %w", err)
	}

	metadata, err := traverse.ParseEDMX(strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate code
	fmt.Printf("Generating code for %d entity types...\n", len(metadata.EntityTypes))
	gen := NewGenerator(metadata, pkgName)

	// Generate types.go
	typesCode := gen.GenerateTypes()
	if err := writeFile(filepath.Join(outputDir, "types.go"), typesCode); err != nil {
		return fmt.Errorf("failed to write types.go: %w", err)
	}
	fmt.Println("✓ Generated types.go")

	// Generate client.go
	clientCode := gen.GenerateClient()
	if err := writeFile(filepath.Join(outputDir, "client.go"), clientCode); err != nil {
		return fmt.Errorf("failed to write client.go: %w", err)
	}
	fmt.Println("✓ Generated client.go")

	// Generate queries.go
	queriesCode := gen.GenerateQueries()
	if err := writeFile(filepath.Join(outputDir, "queries.go"), queriesCode); err != nil {
		return fmt.Errorf("failed to write queries.go: %w", err)
	}
	fmt.Println("✓ Generated queries.go")

	return nil
}

func writeFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func printUsage() {
	usage := `traverse-gen - Generate typed OData clients from metadata

Usage: traverse-gen -url=<metadata_url> -package=<pkg_name> -output=<output_dir>

Flags:
  -url string
        OData metadata URL (e.g., https://api.example.com/odata/v4/$metadata)
  -package string
        Output package name (e.g., 'generated')
  -output string
        Output directory for generated files
  -help
        Show this help message

Example:
  traverse-gen -url=https://services.odata.org/OData/OData.svc/$metadata \
               -package=odata \
               -output=./generated

The generated code includes:
  - types.go: Entity type definitions with JSON tags
  - client.go: Client wrapper for OData service
  - queries.go: Query builders and CRUD methods
`
	fmt.Fprint(os.Stderr, usage)
}
