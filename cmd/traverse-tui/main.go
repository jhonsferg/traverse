// Command traverse-tui provides an interactive terminal interface for building
// and executing OData queries.
//
// Usage:
//
//	traverse-tui --base-url https://services.odata.org/V4/Northwind/Northwind.svc
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const banner = `
 ████████╗██████╗  █████╗ ██╗   ██╗███████╗██████╗ ███████╗███████╗
    ██╔══╝██╔══██╗██╔══██╗██║   ██║██╔════╝██╔══██╗██╔════╝██╔════╝
    ██║   ██████╔╝███████║██║   ██║█████╗  ██████╔╝███████╗█████╗  
    ██║   ██╔══██╗██╔══██║╚██╗ ██╔╝██╔══╝  ██╔══██╗╚════██║██╔══╝  
    ██║   ██║  ██║██║  ██║ ╚████╔╝ ███████╗██║  ██║███████║███████╗
    ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝╚══════╝╚══════╝
                     OData Interactive Query Builder
`

type session struct {
	baseURL    string
	entitySet  string
	filters    []string
	selects    []string
	orderBy    string
	top        int
	skip       int
	httpClient *http.Client
}

func main() {
	baseURL := flag.String("base-url", "", "OData service base URL")
	flag.Parse()

	fmt.Print(banner)

	s := &session{
		baseURL:    *baseURL,
		httpClient: &http.Client{},
		top:        10,
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Type 'help' for available commands, 'exit' to quit.")
	fmt.Println()

	for {
		prompt(s)
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if !handleCommand(s, line) {
			break
		}
	}

	fmt.Println("\nGoodbye!")
}

func prompt(s *session) {
	entity := s.entitySet
	if entity == "" {
		entity = "<no entity set>"
	}
	fmt.Printf("traverse[%s]> ", entity)
}

func handleCommand(s *session, line string) bool {
	parts := strings.SplitN(line, " ", 2)
	cmd := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}

	switch cmd {
	case "exit", "quit", "q":
		return false

	case "help", "h", "?":
		printHelp()

	case "use":
		if arg == "" {
			fmt.Println("usage: use <EntitySet>")
		} else {
			s.entitySet = arg
			s.filters = nil
			s.selects = nil
			s.orderBy = ""
			fmt.Printf("Using entity set: %s\n", arg)
		}

	case "url":
		if arg == "" {
			fmt.Println("usage: url <base-url>")
		} else {
			s.baseURL = arg
			fmt.Printf("Base URL set to: %s\n", arg)
		}

	case "filter", "where":
		if arg == "" {
			fmt.Println("usage: filter <expression>  e.g.: filter Country eq 'Germany'")
		} else {
			s.filters = append(s.filters, arg)
			fmt.Printf("Filter added: %s\n", arg)
		}

	case "select":
		if arg == "" {
			fmt.Println("usage: select <field1,field2,...>")
		} else {
			s.selects = strings.Split(arg, ",")
			fmt.Printf("Select: %s\n", arg)
		}

	case "orderby":
		s.orderBy = arg
		fmt.Printf("Order by: %s\n", arg)

	case "top":
		n := 0
		if _, err := fmt.Sscan(arg, &n); err == nil && n > 0 {
			s.top = n
			fmt.Printf("Top: %d\n", n)
		}

	case "skip":
		n := 0
		if _, err := fmt.Sscan(arg, &n); err == nil {
			s.skip = n
			fmt.Printf("Skip: %d\n", n)
		}

	case "reset", "clear":
		s.filters = nil
		s.selects = nil
		s.orderBy = ""
		s.top = 10
		s.skip = 0
		fmt.Println("Query reset.")

	case "show", "query":
		fmt.Println(buildURL(s))

	case "run", "exec", "go":
		runQuery(s)

	case "status":
		printStatus(s)

	default:
		fmt.Printf("Unknown command: %q. Type 'help' for available commands.\n", cmd)
	}

	return true
}

func buildURL(s *session) string {
	if s.baseURL == "" || s.entitySet == "" {
		return "(incomplete: need base-url and entity set)"
	}
	u := strings.TrimRight(s.baseURL, "/") + "/" + s.entitySet

	var params []string
	if len(s.filters) > 0 {
		params = append(params, "$filter="+strings.Join(s.filters, " and "))
	}
	if len(s.selects) > 0 {
		params = append(params, "$select="+strings.Join(s.selects, ","))
	}
	if s.orderBy != "" {
		params = append(params, "$orderby="+s.orderBy)
	}
	if s.top > 0 {
		params = append(params, fmt.Sprintf("$top=%d", s.top))
	}
	if s.skip > 0 {
		params = append(params, fmt.Sprintf("$skip=%d", s.skip))
	}
	if len(params) > 0 {
		u += "?" + strings.Join(params, "&")
	}
	return u
}

func runQuery(s *session) {
	url := buildURL(s)
	if strings.HasPrefix(url, "(") {
		fmt.Println(url)
		return
	}

	fmt.Printf("GET %s\n", url)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	fmt.Printf("Status: %s\n", resp.Status)

	// Read and pretty-print first 2KB of response
	buf := make([]byte, 2048)
	n, _ := resp.Body.Read(buf)
	if n > 0 {
		fmt.Println(string(buf[:n]))
		if n == 2048 {
			fmt.Println("... (truncated, use $top to limit results)")
		}
	}
}

func printStatus(s *session) {
	fmt.Printf("Base URL:   %s\n", s.baseURL)
	fmt.Printf("Entity Set: %s\n", s.entitySet)
	fmt.Printf("Filters:    %v\n", s.filters)
	fmt.Printf("Select:     %v\n", s.selects)
	fmt.Printf("Order By:   %s\n", s.orderBy)
	fmt.Printf("Top:        %d\n", s.top)
	fmt.Printf("Skip:       %d\n", s.skip)
	fmt.Printf("Query URL:  %s\n", buildURL(s))
}

func printHelp() {
	fmt.Print(`
Commands:
  url <base-url>          Set the OData service base URL
  use <EntitySet>         Select an entity set to query (resets filters)
  filter <expr>           Add a $filter expression (stackable)
  select <fields>         Set $select fields (comma-separated)
  orderby <expr>          Set $orderby expression
  top <n>                 Set $top (default: 10)
  skip <n>                Set $skip (default: 0)
  show / query            Show the current query URL
  run / exec / go         Execute the current query
  status                  Show current session state
  reset / clear           Reset all query options
  help / h / ?            Show this help
  exit / quit / q         Exit

Examples:
  url https://services.odata.org/V4/Northwind/Northwind.svc
  use Customers
  filter Country eq 'Germany'
  select CustomerID,CompanyName,Country
  top 5
  run
`)
}
