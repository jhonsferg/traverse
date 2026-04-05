package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jhonsferg/traverse"
)

func runActions(args []string) {
	fs := flag.NewFlagSet("actions", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: traverse actions [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	urlFlag := fs.String("url", "", "OData service URL (required)")
	userFlag := fs.String("user", "", "Username for basic authentication")
	passFlag := fs.String("pass", "", "Password for basic authentication")
	tokenFlag := fs.String("token", "", "Bearer token for authentication")
	profileFlag := fs.String("profile", "", "Use a saved connection profile")
	formatFlag := fs.String("format", "text", "Output format: json, text")
	timeoutFlag := fs.Int("timeout", 30, "Request timeout in seconds")

	_ = fs.Parse(args)

	conn, err := getConnection(*urlFlag, *userFlag, *passFlag, *tokenFlag, *profileFlag, *timeoutFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = actionsCommand(conn, *formatFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func actionsCommand(conn *Connection, format string) error {
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

	switch format {
	case "json":
		return formatActionsJSON(metadata)
	default:
		return formatActionsText(metadata)
	}
}

func formatActionsText(metadata *traverse.Metadata) error {
	fmt.Printf("OData Service Actions and Functions\n")
	fmt.Printf("====================================\n\n")

	if len(metadata.FunctionImports) > 0 {
		fmt.Printf("Function Imports (OData v2):\n")
		fmt.Printf("%s\n", "---")
		for _, fi := range metadata.FunctionImports {
			fmt.Printf("  %s\n", fi.Name)
			fmt.Printf("    Return Type: %s\n", fi.ReturnType)
			if len(fi.Parameters) > 0 {
				fmt.Printf("    Parameters:\n")
				for _, param := range fi.Parameters {
					nullable := ""
					if param.Nullable {
						nullable = " (nullable)"
					}
					fmt.Printf("      - %s: %s%s\n", param.Name, param.Type, nullable)
				}
			}
		}
		fmt.Printf("\n")
	}

	if len(metadata.Functions) > 0 {
		fmt.Printf("Functions (OData v4):\n")
		fmt.Printf("%s\n", "---")
		for _, fn := range metadata.Functions {
			fmt.Printf("  %s\n", fn.Name)
			fmt.Printf("    Return Type: %s\n", fn.ReturnType)
			if len(fn.Parameters) > 0 {
				fmt.Printf("    Parameters:\n")
				for _, param := range fn.Parameters {
					fmt.Printf("      - %s: %s\n", param.Name, param.Type)
				}
			}
		}
		fmt.Printf("\n")
	}

	if len(metadata.Actions) > 0 {
		fmt.Printf("Actions (OData v4):\n")
		fmt.Printf("%s\n", "---")
		for _, action := range metadata.Actions {
			fmt.Printf("  %s\n", action.Name)
			fmt.Printf("    Return Type: %s\n", action.ReturnType)
			if len(action.Parameters) > 0 {
				fmt.Printf("    Parameters:\n")
				for _, param := range action.Parameters {
					fmt.Printf("      - %s: %s\n", param.Name, param.Type)
				}
			}
		}
		fmt.Printf("\n")
	}

	totalActions := len(metadata.FunctionImports) + len(metadata.Functions) + len(metadata.Actions)
	if totalActions == 0 {
		fmt.Println("No actions or functions available in this service.")
	}

	return nil
}

func formatActionsJSON(metadata *traverse.Metadata) error {
	output := map[string]interface{}{
		"function_imports": metadata.FunctionImports,
		"functions":        metadata.Functions,
		"actions":          metadata.Actions,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}
