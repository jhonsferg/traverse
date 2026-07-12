package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jhonsferg/traverse"
)

type Connection struct {
	URL      string
	Username string
	Password string
	Token    string
	Timeout  int
}

type QueryOptions struct {
	Filter  string
	Select  string
	OrderBy string
	Skip    int
	Top     int
}

type ExportOptions struct {
	Format string
	Output string
	Filter string
	Select string
	Limit  int
}

func metadataCommand(conn *Connection, format string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conn.Timeout)*time.Second)
	defer cancel()

	client, err := createClient(conn)
	if err != nil {
		return err
	}

	metadata, err := client.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	entitySets := metadata.EntitySets

	switch format {
	case "json":
		var data []byte
		data, err = json.MarshalIndent(entitySets, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	default:
		fmt.Printf("Available Entities: %d\n\n", len(entitySets))
		for _, entitySet := range entitySets {
			fmt.Printf("  %s\n", entitySet.Name)
		}
	}

	return nil
}

func describeCommand(conn *Connection, entityName, format string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conn.Timeout)*time.Second)
	defer cancel()

	client, err := createClient(conn)
	if err != nil {
		return err
	}

	metadata, err := client.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	// Find entity set
	var entitySet *traverse.EntitySetInfo
	for i := range metadata.EntitySets {
		if metadata.EntitySets[i].Name == entityName {
			entitySet = &metadata.EntitySets[i]
			break
		}
	}
	if entitySet == nil {
		return fmt.Errorf("entity '%s' not found", entityName)
	}

	// Find entity type
	var entityType *traverse.EntityType
	for i := range metadata.EntityTypes {
		if metadata.EntityTypes[i].Name == entitySet.EntityTypeName {
			entityType = &metadata.EntityTypes[i]
			break
		}
	}
	if entityType == nil {
		return fmt.Errorf("entity type '%s' not found", entitySet.EntityTypeName)
	}

	switch format {
	case "json":
		data := map[string]interface{}{
			"name":       entitySet.Name,
			"type":       entitySet.EntityTypeName,
			"properties": entityType.Properties,
		}
		var output []byte
		output, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(output))
	default:
		fmt.Printf("Entity: %s\n", entitySet.Name)
		fmt.Printf("Type: %s\n\n", entitySet.EntityTypeName)
		fmt.Println("Properties:")
		for _, prop := range entityType.Properties {
			nullable := "required"
			if prop.Nullable {
				nullable = "nullable"
			}
			fmt.Printf("  %-30s %-20s %-15s\n", prop.Name, prop.Type, nullable)
		}
	}

	return nil
}

func countCommand(conn *Connection, entityName, filter string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conn.Timeout)*time.Second)
	defer cancel()

	client, err := createClient(conn)
	if err != nil {
		return err
	}

	query := client.From(entityName)
	if filter != "" {
		query = query.Filter(filter)
	}

	totalCount, countErr := query.Count(ctx)
	if countErr != nil {
		return fmt.Errorf("failed to count records: %w", countErr)
	}

	fmt.Printf("Count: %d\n", totalCount)
	return nil
}

func sampleCommand(conn *Connection, entityName string, count int, filter, selectFields, format string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conn.Timeout)*time.Second)
	defer cancel()

	client, err := createClient(conn)
	if err != nil {
		return err
	}

	query := client.From(entityName).Top(count)
	if filter != "" {
		query = query.Filter(filter)
	}
	if selectFields != "" {
		props := strings.Split(selectFields, ",")
		for i := range props {
			props[i] = strings.TrimSpace(props[i])
		}
		query = query.Select(props...)
	}

	outputResults, execErr := query.Collect(ctx)
	if execErr != nil {
		return fmt.Errorf("query execution failed: %w", execErr)
	}

	return formatOutput(outputResults, format)

}

func queryCommand(conn *Connection, entityName string, opts QueryOptions, format string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conn.Timeout)*time.Second)
	defer cancel()

	client, err := createClient(conn)
	if err != nil {
		return err
	}

	query := client.From(entityName)

	if opts.Filter != "" {
		query = query.Filter(opts.Filter)
	}
	if opts.Select != "" {
		props := strings.Split(opts.Select, ",")
		for i := range props {
			props[i] = strings.TrimSpace(props[i])
		}
		query = query.Select(props...)
	}
	if opts.OrderBy != "" {
		query = query.OrderBy(opts.OrderBy)
	}
	if opts.Skip > 0 {
		query = query.Skip(opts.Skip)
	}
	if opts.Top > 0 {
		query = query.Top(opts.Top)
	}

	outputResults, execErr := query.Collect(ctx)
	if execErr != nil {
		return fmt.Errorf("query execution failed: %w", execErr)
	}

	return formatOutput(outputResults, format)

}

func exportCommand(conn *Connection, entityName string, opts ExportOptions) error {
	if opts.Output == "" {
		return fmt.Errorf("output file path is required")
	}

	if opts.Format != "csv" && opts.Format != "json" {
		return fmt.Errorf("unsupported format: %s (must be csv or json)", opts.Format)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conn.Timeout)*time.Second)
	defer cancel()

	client, err := createClient(conn)
	if err != nil {
		return err
	}

	query := client.From(entityName)
	if opts.Filter != "" {
		query = query.Filter(opts.Filter)
	}
	if opts.Select != "" {
		props := strings.Split(opts.Select, ",")
		for i := range props {
			props[i] = strings.TrimSpace(props[i])
		}
		query = query.Select(props...)
	}
	if opts.Limit > 0 {
		query = query.Top(opts.Limit)
	}

	exportData, fetchErr := query.Collect(ctx)
	if fetchErr != nil {
		return fmt.Errorf("failed to fetch data: %w", fetchErr)
	}

	outputFile, createErr := os.Create(opts.Output)
	if createErr != nil {
		return fmt.Errorf("failed to create output file: %w", createErr)
	}
	defer func() {
		if closeErr := outputFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close file %s: %v\n", opts.Output, closeErr)
		}
	}()

	switch opts.Format {
	case "json":
		var data []byte
		data, err = json.MarshalIndent(exportData, "", "  ")
		if err != nil {
			return err
		}
		_, err = outputFile.WriteString(string(data))
		if err != nil {
			return err
		}
	case "csv":
		err = exportToCSV(outputFile, exportData)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Data exported to %s\n", opts.Output)
	return nil
}

func createClient(conn *Connection) (*traverse.Client, error) {
	// Ensure base URL ends with / for proper RFC 3986 path resolution in Relay
	baseURL := conn.URL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	opts := []traverse.Option{
		traverse.WithBaseURL(baseURL),
	}

	if conn.Token != "" {
		opts = append(opts, traverse.WithBearerToken(conn.Token))
	} else if conn.Username != "" && conn.Password != "" {
		opts = append(opts, traverse.WithBasicAuth(conn.Username, conn.Password))
	}

	client, err := traverse.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}

func formatOutput(data []map[string]interface{}, format string) error {
	// "table" is the only format with its own renderer. Every other value -
	// including "json", "text" (several commands default --format to
	// "text"), "" (unset), and any unrecognized/typo'd value - intentionally
	// falls back to JSON output: this CLI has no separate plain-text
	// renderer, and it stays forgiving of typos rather than erroring out.
	if format == "table" {
		return formatTable(data)
	}

	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func formatTable(data []map[string]interface{}) error {
	if len(data) == 0 {
		fmt.Println("No records found")
		return nil
	}

	// Extract column names from first row
	firstRow := data[0]
	columns := make([]string, 0, len(firstRow))
	for k := range firstRow {
		columns = append(columns, k)
	}
	sort.Strings(columns)

	// Calculate column widths
	widths := make(map[string]int)
	for _, col := range columns {
		widths[col] = len(col)
	}

	for _, row := range data {
		for _, col := range columns {
			val := toString(row[col])
			if len(val) > widths[col] {
				widths[col] = len(val)
			}
		}
	}

	// Print header
	for i, col := range columns {
		fmt.Printf("%-*s", widths[col]+2, col)
		if i < len(columns)-1 {
			fmt.Print(" | ")
		}
	}
	fmt.Println()

	// Print separator
	for i, col := range columns {
		fmt.Print(strings.Repeat("-", widths[col]+2))
		if i < len(columns)-1 {
			fmt.Print("-+-")
		}
	}
	fmt.Println()

	// Print rows
	for _, row := range data {
		for i, col := range columns {
			val := toString(row[col])
			fmt.Printf("%-*s", widths[col]+2, val)
			if i < len(columns)-1 {
				fmt.Print(" | ")
			}
		}
		fmt.Println()
	}

	fmt.Printf("\nTotal: %d records\n", len(data))
	return nil
}

func toString(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case json.Number:
		return val.String()
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

func exportToCSV(file *os.File, data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Extract column names from first row
	firstRow := data[0]
	columns := make([]string, 0, len(firstRow))
	for k := range firstRow {
		columns = append(columns, k)
	}
	sort.Strings(columns)

	// Write header
	if err := writer.Write(columns); err != nil {
		return err
	}

	// Write rows
	for _, row := range data {
		record := make([]string, 0, len(columns))
		for _, col := range columns {
			record = append(record, toString(row[col]))
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
