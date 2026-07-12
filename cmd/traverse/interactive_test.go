package main

import (
	"bufio"
	"strings"
	"testing"
)

func TestPrintInteractiveHelp(t *testing.T) {
	// Just verifies it doesn't panic; output goes to stdout.
	printInteractiveHelp()
}

func TestInteractiveConnect_EmptyURL(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	conn, err := interactiveConnect(reader)
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	if conn != nil {
		t.Fatalf("expected nil connection, got %+v", conn)
	}
}

func TestInteractiveConnect_UnreachableHost(t *testing.T) {
	// Port 1 on localhost should refuse the connection immediately without
	// touching the network, exercising the "failed to create client /
	// failed to connect" error path deterministically and fast.
	input := "http://127.0.0.1:1/\n\n\n\n"
	reader := bufio.NewReader(strings.NewReader(input))
	conn, err := interactiveConnect(reader)
	if err == nil {
		t.Fatal("expected error connecting to an unreachable host")
	}
	if conn != nil {
		t.Fatalf("expected nil connection, got %+v", conn)
	}
}

func TestInteractiveQuery_EmptyEntity(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	err := interactiveQuery(reader, &Connection{URL: "http://example.com/"})
	if err == nil {
		t.Fatal("expected error for empty entity name")
	}
}

func TestInteractiveExport_EmptyEntity(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	err := interactiveExport(reader, &Connection{URL: "http://example.com/"})
	if err == nil {
		t.Fatal("expected error for empty entity name")
	}
}

func TestInteractiveExport_EmptyOutputPath(t *testing.T) {
	input := "Product\n\n"
	reader := bufio.NewReader(strings.NewReader(input))
	err := interactiveExport(reader, &Connection{URL: "http://example.com/"})
	if err == nil {
		t.Fatal("expected error for empty output path")
	}
}

func TestInteractiveExport_InvalidFormat(t *testing.T) {
	input := "Product\n/tmp/out.xml\nxml\n"
	reader := bufio.NewReader(strings.NewReader(input))
	err := interactiveExport(reader, &Connection{URL: "http://example.com/"})
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestInteractiveExport_DefaultFormat(t *testing.T) {
	// Blank format line should default to "json" and pass validation
	// (it will only fail later, once exportCommand tries to reach conn.URL).
	input := "Product\n/tmp/out.json\n\n\n\n\n"
	reader := bufio.NewReader(strings.NewReader(input))
	err := interactiveExport(reader, &Connection{URL: "http://127.0.0.1:1/"})
	// We expect this to fail (unreachable host), but NOT with "invalid format".
	if err == nil {
		t.Fatal("expected an error due to unreachable connection")
	}
	if strings.Contains(err.Error(), "invalid format") {
		t.Fatalf("default format should be valid, got: %v", err)
	}
}
