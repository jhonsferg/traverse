package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

func startInteractive() error {
	var currentConn *Connection
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Traverse - OData CLI Explorer (Interactive Mode)")
	fmt.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println()

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]

		switch command {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return nil

		case "help":
			printInteractiveHelp()

		case "connect":
			conn, err := interactiveConnect(reader)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			currentConn = conn
			fmt.Printf("Connected to: %s\n", currentConn.URL)

		case "entities":
			if currentConn == nil {
				fmt.Println("Error: Not connected. Use 'connect' first.")
				continue
			}
			err := interactiveMetadata(currentConn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}

		case "describe":
			if currentConn == nil {
				fmt.Println("Error: Not connected. Use 'connect' first.")
				continue
			}
			if len(parts) < 2 {
				fmt.Println("Usage: describe <entity_name>")
				continue
			}
			err := describeCommand(currentConn, parts[1], "text")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}

		case "count":
			if currentConn == nil {
				fmt.Println("Error: Not connected. Use 'connect' first.")
				continue
			}
			if len(parts) < 2 {
				fmt.Println("Usage: count <entity_name> [filter]")
				continue
			}
			filter := ""
			if len(parts) > 2 {
				filter = strings.Join(parts[2:], " ")
			}
			err := countCommand(currentConn, parts[1], filter)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}

		case "sample":
			if currentConn == nil {
				fmt.Println("Error: Not connected. Use 'connect' first.")
				continue
			}
			if len(parts) < 2 {
				fmt.Println("Usage: sample <entity_name> [count]")
				continue
			}
			count := 5
			if len(parts) > 2 {
				var n int
				n, err = fmt.Sscanf(parts[2], "%d", &count)
				if err != nil || n != 1 {
					fmt.Fprintf(os.Stderr, "Warning: invalid count: %v\n", err)
					count = 5 // use default
				}
			}
			err := sampleCommand(currentConn, parts[1], count, "", "", "table")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}

		case "query":
			if currentConn == nil {
				fmt.Println("Error: Not connected. Use 'connect' first.")
				continue
			}
			err := interactiveQuery(reader, currentConn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}

		case "export":
			if currentConn == nil {
				fmt.Println("Error: Not connected. Use 'connect' first.")
				continue
			}
			err := interactiveExport(reader, currentConn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}

		case "disconnect":
			currentConn = nil
			fmt.Println("Disconnected")

		case "status":
			if currentConn == nil {
				fmt.Println("Status: Not connected")
			} else {
				fmt.Printf("Status: Connected\n")
				fmt.Printf("URL: %s\n", currentConn.URL)
				fmt.Printf("Timeout: %d seconds\n", currentConn.Timeout)
			}

		case "clear":
			fmt.Print("\033[2J\033[H")

		default:
			fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", command)
		}
	}
}

func printInteractiveHelp() {
	fmt.Print(`
Available Commands:

  connect              Connect to an OData service (interactive prompts)
  disconnect           Disconnect from current service
  entities             List all available entities
  describe <entity>    Show entity structure and properties
  count <entity>       Count records in an entity
  sample <entity>      Show sample records from an entity
  query                Build and execute a custom OData query (interactive)
  export               Export data to CSV or JSON (interactive)
  status               Show current connection status
  clear                Clear the screen
  help                 Show this help message
  exit, quit           Exit the application

Examples:
  > connect
  > entities
  > describe Product
  > sample Product 10
  > count Product
  > query
  > export

`)
	fmt.Println()
}

func interactiveConnect(reader *bufio.Reader) (*Connection, error) {
	fmt.Print("OData Service URL: ")
	url, _ := reader.ReadString('\n')
	url = strings.TrimSpace(url)

	if url == "" {
		return nil, fmt.Errorf("URL is required")
	}

	fmt.Print("Username (leave blank for none): ")
	user, _ := reader.ReadString('\n')
	user = strings.TrimSpace(user)

	fmt.Print("Password (leave blank for none): ")
	pass, _ := reader.ReadString('\n')
	pass = strings.TrimSpace(pass)

	fmt.Print("Bearer token (leave blank for none): ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	conn := &Connection{
		URL:      url,
		Username: user,
		Password: pass,
		Token:    token,
		Timeout:  30,
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := createClient(conn)
	if err != nil {
		return nil, err
	}

	_, err = client.Metadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return conn, nil
}

func interactiveMetadata(conn *Connection) error {
	return metadataCommand(conn, "text")
}

func interactiveQuery(reader *bufio.Reader, conn *Connection) error {
	fmt.Print("Entity name: ")
	entity, _ := reader.ReadString('\n')
	entity = strings.TrimSpace(entity)

	if entity == "" {
		return fmt.Errorf("entity name is required")
	}

	fmt.Print("Filter (OData filter syntax, leave blank for none): ")
	filter, _ := reader.ReadString('\n')
	filter = strings.TrimSpace(filter)

	fmt.Print("Select properties (comma-separated, leave blank for all): ")
	select_, _ := reader.ReadString('\n')
	select_ = strings.TrimSpace(select_)

	fmt.Print("Order by (property name, leave blank for none): ")
	orderby, _ := reader.ReadString('\n')
	orderby = strings.TrimSpace(orderby)

	fmt.Print("Skip count (0 for none): ")
	skipStr, _ := reader.ReadString('\n')
	skip := 0
	if n, err := fmt.Sscanf(strings.TrimSpace(skipStr), "%d", &skip); err != nil || n != 1 {
		skip = 0 // use default
	}

	fmt.Print("Top count (0 for all): ")
	topStr, _ := reader.ReadString('\n')
	top := 0
	if n, err := fmt.Sscanf(strings.TrimSpace(topStr), "%d", &top); err != nil || n != 1 {
		top = 0 // use default
	}

	fmt.Print("Output format (json, table, text) [default: table]: ")
	format, _ := reader.ReadString('\n')
	format = strings.TrimSpace(format)
	if format == "" {
		format = "table"
	}

	opts := QueryOptions{
		Filter:  filter,
		Select:  select_,
		OrderBy: orderby,
		Skip:    skip,
		Top:     top,
	}

	return queryCommand(conn, entity, opts, format)
}

func interactiveExport(reader *bufio.Reader, conn *Connection) error {
	fmt.Print("Entity name: ")
	entity, _ := reader.ReadString('\n')
	entity = strings.TrimSpace(entity)

	if entity == "" {
		return fmt.Errorf("entity name is required")
	}

	fmt.Print("Output file path: ")
	output, _ := reader.ReadString('\n')
	output = strings.TrimSpace(output)

	if output == "" {
		return fmt.Errorf("output file path is required")
	}

	fmt.Print("Format (csv, json) [default: json]: ")
	format, _ := reader.ReadString('\n')
	format = strings.TrimSpace(format)
	if format == "" {
		format = "json"
	}

	if format != "csv" && format != "json" {
		return fmt.Errorf("invalid format: %s", format)
	}

	fmt.Print("Filter (OData filter syntax, leave blank for none): ")
	filter, _ := reader.ReadString('\n')
	filter = strings.TrimSpace(filter)

	fmt.Print("Select properties (comma-separated, leave blank for all): ")
	select_, _ := reader.ReadString('\n')
	select_ = strings.TrimSpace(select_)

	fmt.Print("Limit records (0 for no limit): ")
	limitStr, _ := reader.ReadString('\n')
	limit := 0
	if n, err := fmt.Sscanf(strings.TrimSpace(limitStr), "%d", &limit); err != nil || n != 1 {
		limit = 0 // use default
	}

	opts := ExportOptions{
		Format: format,
		Output: output,
		Filter: filter,
		Select: select_,
		Limit:  limit,
	}

	return exportCommand(conn, entity, opts)
}
