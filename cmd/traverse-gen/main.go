package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var (
		metadataURL  = flag.String("metadata-url", "", "OData $metadata URL (e.g. https://api.example.com/odata/v4/$metadata)")
		metadataFile = flag.String("metadata-file", "", "Local EDMX XML file to read instead of fetching from a URL")
		outputDir    = flag.String("output-dir", "generated", "Output directory for generated files (default: ./generated)")
		pkgName      = flag.String("package-name", "odata", "Go package name for generated files (default: odata)")
		namespace    = flag.String("namespace", "", "OData namespace to process; empty means all namespaces")
		help         = flag.Bool("help", false, "Show help and exit")
	)
	flag.Parse()

	if *help {
		printUsage()
		os.Exit(0)
	}

	if *metadataURL == "" && *metadataFile == "" {
		fmt.Fprintln(os.Stderr, "Error: one of --metadata-url or --metadata-file is required")
		printUsage()
		os.Exit(1)
	}
	if *metadataURL != "" && *metadataFile != "" {
		fmt.Fprintln(os.Stderr, "Error: --metadata-url and --metadata-file are mutually exclusive")
		os.Exit(1)
	}

	if err := run(*metadataURL, *metadataFile, *outputDir, *pkgName, *namespace); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated OData client in %s\n", *outputDir)
}

func run(metadataURL, metadataFile, outputDir, pkgName, namespace string) error {
	reader, closeFn, err := openMetadata(metadataURL, metadataFile)
	if err != nil {
		return err
	}
	defer closeFn()

	schemas, err := parseEDMX(reader, namespace)
	if err != nil {
		return fmt.Errorf("failed to parse EDMX: %w", err)
	}
	if len(schemas) == 0 {
		return fmt.Errorf("no schemas found in metadata document (namespace filter: %q)", namespace)
	}

	// Count entity types and sets for the progress message.
	totalTypes, totalSets := 0, 0
	for _, s := range schemas {
		totalTypes += len(s.EntityTypes)
		totalSets += len(s.EntitySets)
	}
	fmt.Printf("Parsed %d entity type(s) and %d entity set(s) from %d schema(s)\n",
		totalTypes, totalSets, len(schemas))

	if err = os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	gen := NewCodeGenerator(schemas, pkgName)

	typesCode, err := gen.RenderTypes()
	if err != nil {
		return fmt.Errorf("failed to render types.go: %w", err)
	}
	if err = writeFile(filepath.Join(outputDir, "types.go"), typesCode); err != nil {
		return fmt.Errorf("failed to write types.go: %w", err)
	}
	fmt.Println("Generated types.go")

	clientCode, err := gen.RenderClient()
	if err != nil {
		return fmt.Errorf("failed to render client.go: %w", err)
	}
	if err = writeFile(filepath.Join(outputDir, "client.go"), clientCode); err != nil {
		return fmt.Errorf("failed to write client.go: %w", err)
	}
	fmt.Println("Generated client.go")

	queriesCode, err := gen.RenderQueries()
	if err != nil {
		return fmt.Errorf("failed to render queries.go: %w", err)
	}
	if err = writeFile(filepath.Join(outputDir, "queries.go"), queriesCode); err != nil {
		return fmt.Errorf("failed to write queries.go: %w", err)
	}
	fmt.Println("Generated queries.go")

	return nil
}

// openMetadata returns a reader for the EDMX metadata, either from a URL or a file.
// The caller must invoke the returned close function when done.
func openMetadata(metadataURL, metadataFile string) (io.Reader, func(), error) {
	if metadataFile != "" {
		f, err := os.Open(metadataFile) //nolint:gosec // user-specified path is intentional
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open metadata file: %w", err)
		}
		return f, func() { _ = f.Close() }, nil
	}

	fmt.Printf("Fetching metadata from %s\n", metadataURL)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec,bodyclose // body is returned to caller via closeFn
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, nil, fmt.Errorf("metadata request failed with HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return resp.Body, func() { _ = resp.Body.Close() }, nil
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0600)
}

func printUsage() {
	usage := `traverse-gen - Generate typed OData clients from EDMX metadata

Usage:
  traverse-gen --metadata-url URL  [options]
  traverse-gen --metadata-file FILE [options]

Source flags (exactly one required):
  --metadata-url URL    Fetch $metadata from a live OData service endpoint
  --metadata-file FILE  Read EDMX XML from a local file

Output flags:
  --output-dir DIR      Output directory for generated files (default: generated)
  --package-name NAME   Go package name for generated files (default: odata)
  --namespace NS        OData namespace to process; empty means all namespaces

Other:
  --help                Show this message

Generated files:
  types.go    - OData entity/complex/enum types as Go structs
  client.go   - GeneratedClient wrapper with typed entity set accessors
  queries.go  - Typed query builder structs (one per entity set)

Example:
  traverse-gen --metadata-url https://services.odata.org/V4/Northwind/Northwind.svc/$metadata \
               --output-dir ./northwind \
               --package-name northwind
`
	fmt.Fprint(os.Stderr, usage)
}
