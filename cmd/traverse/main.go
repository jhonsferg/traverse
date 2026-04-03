package main

import (
	"flag"
	"fmt"
	"os"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "help", "-h", "--help":
		printHelp()
	case "version", "-v", "--version":
		fmt.Printf("traverse %s\n", version)
	case "interactive":
		runInteractive()
	case "metadata":
		runMetadata(os.Args[2:])
	case "describe":
		runDescribe(os.Args[2:])
	case "count":
		runCount(os.Args[2:])
	case "sample":
		runSample(os.Args[2:])
	case "query":
		runQuery(os.Args[2:])
	case "export":
		runExport(os.Args[2:])
	case "profile":
		runProfile(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: traverse <command> [options]

Commands:
  metadata    List all entity sets in the OData service
  describe    Show the structure of an entity
  count       Count records in an entity
  sample      Show N sample records from an entity
  query       Build and execute custom OData queries
  export      Export data to CSV or JSON
  profile     Manage connection profiles
  interactive Start interactive exploration mode
  help        Show this help message
  version     Show version information

Use "traverse <command> -h" for command-specific help.
Use "traverse interactive" to start the interactive explorer.

Example:
  traverse metadata -url https://api.example.com/odata
  traverse describe -url https://api.example.com/odata -entity Product

`)
}

func printHelp() {
	fmt.Printf(`Traverse - OData CLI Explorer v%s

Traverse helps you explore and query OData services from the command line.

Usage: traverse <command> [options]

Global Options:
  -url        OData service URL
  -user       Username for basic authentication
  -pass       Password for basic authentication
  -token      Bearer token for authentication
  -profile    Use a saved connection profile
  -timeout    Request timeout in seconds (default: 30)

Commands:

  metadata [options]
    Lists all available entity sets in the OData service.
    Options:
      -url string       OData service URL (required)
      -format string    Output format: json, text (default: text)

  describe [options]
    Shows the structure and properties of an entity.
    Options:
      -url string       OData service URL (required)
      -entity string    Entity name (required)
      -format string    Output format: json, text (default: text)

  count [options]
    Counts the number of records in an entity.
    Options:
      -url string       OData service URL (required)
      -entity string    Entity name (required)
      -filter string    OData filter expression
      -timeout int      Request timeout in seconds (default: 30)

  sample [options]
    Retrieves N sample records from an entity.
    Options:
      -url string       OData service URL (required)
      -entity string    Entity name (required)
      -count int        Number of records to show (default: 5)
      -filter string    OData filter expression
      -select string    Comma-separated property names
      -format string    Output format: json, text, table (default: table)

  query [options]
    Builds and executes custom OData queries.
    Options:
      -url string       OData service URL (required)
      -entity string    Entity name (required)
      -filter string    OData filter expression
      -select string    Comma-separated property names
      -orderby string   Sort specification
      -skip int         Skip N records
      -top int          Take N records
      -format string    Output format: json, text, table (default: table)

  export [options]
    Exports entity data to CSV or JSON file.
    Options:
      -url string       OData service URL (required)
      -entity string    Entity name (required)
      -format string    Output format: csv, json (default: json)
      -output string    Output file path (required)
      -filter string    OData filter expression
      -select string    Comma-separated property names
      -limit int        Maximum records to export (default: no limit)

  profile <action> [options]
    Manages connection profiles.
    Actions:
      list              List all saved profiles
      create            Create a new profile
      delete <name>     Delete a profile
      set-default <name> Set default profile
    Options (for create):
      -name string      Profile name (required)
      -url string       OData service URL (required)
      -user string      Username
      -pass string      Password
      -token string     Bearer token

  interactive
    Starts an interactive mode for exploring OData services.
    In interactive mode, you can:
      - Switch between different OData services
      - Explore entities and their properties
      - Run queries interactively
      - See results in formatted tables
      - Use readline for command history and editing

Examples:

  # List all entities in an OData service
  traverse metadata -url https://api.example.com/odata

  # Show structure of a Product entity
  traverse describe -url https://api.example.com/odata -entity Product

  # Get 10 sample products
  traverse sample -url https://api.example.com/odata -entity Product -count 10

  # Query with filters
  traverse query -url https://api.example.com/odata -entity Product \
    -filter "Price gt 100" -orderby "Name" -top 5

  # Export all products to JSON
  traverse export -url https://api.example.com/odata -entity Product \
    -output products.json

  # Save a connection profile
  traverse profile create -name myservice -url https://api.example.com/odata

  # Use a profile
  traverse metadata -profile myservice

  # Interactive exploration
  traverse interactive

For more information, visit: https://github.com/jhonsferg/traverse
`, version)
}

func runMetadata(args []string) {
	fs := flag.NewFlagSet("metadata", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: traverse metadata [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	urlFlag := fs.String("url", "", "OData service URL (required)")
	userFlag := fs.String("user", "", "Username for basic authentication")
	passFlag := fs.String("pass", "", "Password for basic authentication")
	tokenFlag := fs.String("token", "", "Bearer token for authentication")
	profileFlag := fs.String("profile", "", "Use a saved connection profile")
	formatFlag := fs.String("format", "text", "Output format: json, text")
	timeoutFlag := fs.Int("timeout", 30, "Request timeout in seconds")

	fs.Parse(args)

	conn, err := getConnection(*urlFlag, *userFlag, *passFlag, *tokenFlag, *profileFlag, *timeoutFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = metadataCommand(conn, *formatFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runDescribe(args []string) {
	fs := flag.NewFlagSet("describe", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: traverse describe [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	urlFlag := fs.String("url", "", "OData service URL (required)")
	entityFlag := fs.String("entity", "", "Entity name (required)")
	userFlag := fs.String("user", "", "Username for basic authentication")
	passFlag := fs.String("pass", "", "Password for basic authentication")
	tokenFlag := fs.String("token", "", "Bearer token for authentication")
	profileFlag := fs.String("profile", "", "Use a saved connection profile")
	formatFlag := fs.String("format", "text", "Output format: json, text")
	timeoutFlag := fs.Int("timeout", 30, "Request timeout in seconds")

	fs.Parse(args)

	conn, err := getConnection(*urlFlag, *userFlag, *passFlag, *tokenFlag, *profileFlag, *timeoutFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = describeCommand(conn, *entityFlag, *formatFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCount(args []string) {
	fs := flag.NewFlagSet("count", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: traverse count [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	urlFlag := fs.String("url", "", "OData service URL (required)")
	entityFlag := fs.String("entity", "", "Entity name (required)")
	filterFlag := fs.String("filter", "", "OData filter expression")
	userFlag := fs.String("user", "", "Username for basic authentication")
	passFlag := fs.String("pass", "", "Password for basic authentication")
	tokenFlag := fs.String("token", "", "Bearer token for authentication")
	profileFlag := fs.String("profile", "", "Use a saved connection profile")
	timeoutFlag := fs.Int("timeout", 30, "Request timeout in seconds")

	fs.Parse(args)

	conn, err := getConnection(*urlFlag, *userFlag, *passFlag, *tokenFlag, *profileFlag, *timeoutFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = countCommand(conn, *entityFlag, *filterFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSample(args []string) {
	fs := flag.NewFlagSet("sample", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: traverse sample [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	urlFlag := fs.String("url", "", "OData service URL (required)")
	entityFlag := fs.String("entity", "", "Entity name (required)")
	countFlag := fs.Int("count", 5, "Number of records to show")
	filterFlag := fs.String("filter", "", "OData filter expression")
	selectFlag := fs.String("select", "", "Comma-separated property names")
	userFlag := fs.String("user", "", "Username for basic authentication")
	passFlag := fs.String("pass", "", "Password for basic authentication")
	tokenFlag := fs.String("token", "", "Bearer token for authentication")
	profileFlag := fs.String("profile", "", "Use a saved connection profile")
	formatFlag := fs.String("format", "table", "Output format: json, text, table")
	timeoutFlag := fs.Int("timeout", 30, "Request timeout in seconds")

	fs.Parse(args)

	conn, err := getConnection(*urlFlag, *userFlag, *passFlag, *tokenFlag, *profileFlag, *timeoutFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = sampleCommand(conn, *entityFlag, *countFlag, *filterFlag, *selectFlag, *formatFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runQuery(args []string) {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: traverse query [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	urlFlag := fs.String("url", "", "OData service URL (required)")
	entityFlag := fs.String("entity", "", "Entity name (required)")
	filterFlag := fs.String("filter", "", "OData filter expression")
	selectFlag := fs.String("select", "", "Comma-separated property names")
	orderbyFlag := fs.String("orderby", "", "Sort specification")
	skipFlag := fs.Int("skip", 0, "Skip N records")
	topFlag := fs.Int("top", 0, "Take N records")
	userFlag := fs.String("user", "", "Username for basic authentication")
	passFlag := fs.String("pass", "", "Password for basic authentication")
	tokenFlag := fs.String("token", "", "Bearer token for authentication")
	profileFlag := fs.String("profile", "", "Use a saved connection profile")
	formatFlag := fs.String("format", "table", "Output format: json, text, table")
	timeoutFlag := fs.Int("timeout", 30, "Request timeout in seconds")

	fs.Parse(args)

	conn, err := getConnection(*urlFlag, *userFlag, *passFlag, *tokenFlag, *profileFlag, *timeoutFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	queryOpts := QueryOptions{
		Filter:  *filterFlag,
		Select:  *selectFlag,
		OrderBy: *orderbyFlag,
		Skip:    *skipFlag,
		Top:     *topFlag,
	}

	err = queryCommand(conn, *entityFlag, queryOpts, *formatFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: traverse export [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	urlFlag := fs.String("url", "", "OData service URL (required)")
	entityFlag := fs.String("entity", "", "Entity name (required)")
	formatFlag := fs.String("format", "json", "Output format: csv, json")
	outputFlag := fs.String("output", "", "Output file path (required)")
	filterFlag := fs.String("filter", "", "OData filter expression")
	selectFlag := fs.String("select", "", "Comma-separated property names")
	limitFlag := fs.Int("limit", 0, "Maximum records to export (0 = no limit)")
	userFlag := fs.String("user", "", "Username for basic authentication")
	passFlag := fs.String("pass", "", "Password for basic authentication")
	tokenFlag := fs.String("token", "", "Bearer token for authentication")
	profileFlag := fs.String("profile", "", "Use a saved connection profile")
	timeoutFlag := fs.Int("timeout", 30, "Request timeout in seconds")

	fs.Parse(args)

	conn, err := getConnection(*urlFlag, *userFlag, *passFlag, *tokenFlag, *profileFlag, *timeoutFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	exportOpts := ExportOptions{
		Format: *formatFlag,
		Output: *outputFlag,
		Filter: *filterFlag,
		Select: *selectFlag,
		Limit:  *limitFlag,
	}

	err = exportCommand(conn, *entityFlag, exportOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runProfile(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: traverse profile <action> [options]\n")
		fmt.Fprintf(os.Stderr, "Actions: list, create, delete, set-default\n")
		os.Exit(1)
	}

	action := args[0]

	switch action {
	case "list":
		profileListCommand()
	case "create":
		fs := flag.NewFlagSet("profile-create", flag.ExitOnError)
		nameFlag := fs.String("name", "", "Profile name (required)")
		urlFlag := fs.String("url", "", "OData service URL (required)")
		userFlag := fs.String("user", "", "Username for basic authentication")
		passFlag := fs.String("pass", "", "Password for basic authentication")
		tokenFlag := fs.String("token", "", "Bearer token for authentication")
		fs.Parse(args[1:])

		err := profileCreateCommand(*nameFlag, *urlFlag, *userFlag, *passFlag, *tokenFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "delete":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: traverse profile delete <name>\n")
			os.Exit(1)
		}
		err := profileDeleteCommand(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "set-default":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: traverse profile set-default <name>\n")
			os.Exit(1)
		}
		err := profileSetDefaultCommand(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown profile action: %s\n", action)
		os.Exit(1)
	}
}

func runInteractive() {
	err := startInteractive()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func getConnection(url, user, pass, token, profile string, timeout int) (*Connection, error) {
	if profile != "" {
		prof, err := loadProfile(profile)
		if err != nil {
			return nil, err
		}
		url = prof.URL
		if prof.Username != "" {
			user = prof.Username
		}
		if prof.Password != "" {
			pass = prof.Password
		}
		if prof.Token != "" {
			token = prof.Token
		}
	}

	if url == "" {
		return nil, fmt.Errorf("OData service URL is required (-url or -profile)")
	}

	conn := &Connection{
		URL:      url,
		Username: user,
		Password: pass,
		Token:    token,
		Timeout:  timeout,
	}

	return conn, nil
}
