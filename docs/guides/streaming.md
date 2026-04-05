# Streaming

traverse streams OData responses over Go channels, allowing you to process arbitrarily large datasets at constant memory. The JSON response is parsed incrementally using a streaming tokenizer - no full response buffering.

## Stream API

```go
stream, err := client.From("Orders").
    Filter("Freight gt 0").
    OrderBy("OrderDate asc").
    Stream(ctx, Order{})
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for item := range stream.Items() {
    order := item.(Order)
    process(order)
}

if err := stream.Err(); err != nil {
    log.Fatal(err)
}
```

## Memory Guarantee

Streaming parses and emits one entity at a time. Memory usage is proportional to a single entity, not the response size. A response with 1,000,000 items uses the same memory as one with 10 items.

This is achieved by:
1. Parsing the HTTP response body as an `io.Reader` (no full body read)
2. Using a streaming JSON tokenizer that emits objects incrementally
3. Sending each decoded entity to the channel before parsing the next

## Closing the Stream

Always call `stream.Close()` when done - this releases the HTTP response body and stops background goroutines:

```go
stream, err := client.From("LargeTable").Stream(ctx, Row{})
if err != nil {
    log.Fatal(err)
}
defer stream.Close() // always defer this

for item := range stream.Items() {
    // process...
    if shouldStop {
        break // closing the channel loop is safe; defer Close() cleans up
    }
}
```

## Error Handling

Check `stream.Err()` after the channel is closed:

```go
for item := range stream.Items() {
    _ = item.(MyType)
}

if err := stream.Err(); err != nil {
    // could be network error, parse error, or context cancellation
    log.Fatal(err)
}
```

The stream closes its channel both on completion and on error.

## Backpressure

The channel has a configurable buffer. If your processing is slower than the network, the channel fills up and the streaming goroutine pauses automatically. You do not need to implement any explicit flow control.

```go
stream, err := client.From("LargeTable").
    StreamWithBuffer(ctx, Row{}, 100) // buffer 100 items
```

Default buffer size is 64.

## Processing 1 Million Records

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/jhonsferg/traverse"
)

type SalesRecord struct {
    ID       string  `json:"SalesID"`
    Region   string  `json:"Region"`
    Amount   float64 `json:"Amount"`
    Date     string  `json:"SaleDate"`
}

func main() {
    client, err := traverse.New(
        traverse.WithBaseURL("https://sap.example.com/sap/opu/odata/sap/SALES_SRV"),
        traverse.WithBasicAuth("user", "pass"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    stream, err := client.From("SalesSet").
        Filter("FiscalYear eq '2024'").
        Select("SalesID", "Region", "Amount", "SaleDate").
        OrderBy("SaleDate asc").
        Stream(ctx, SalesRecord{})
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()

    // Aggregate totals without loading everything into memory
    totals := make(map[string]float64)
    count := 0
    start := time.Now()

    for item := range stream.Items() {
        rec := item.(SalesRecord)
        totals[rec.Region] += rec.Amount
        count++

        if count%10000 == 0 {
            fmt.Printf("Processed %d records...\n", count)
        }
    }

    if err := stream.Err(); err != nil {
        log.Fatal(err)
    }

    elapsed := time.Since(start)
    fmt.Printf("Processed %d records in %s\n", count, elapsed)
    for region, total := range totals {
        fmt.Printf("  %s: %.2f\n", region, total)
    }
}
```

## Streaming with Context Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

stream, err := client.From("LargeSet").Stream(ctx, Row{})
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for item := range stream.Items() {
    _ = item.(Row)
}

if err := stream.Err(); err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("stream timed out")
    } else {
        log.Fatal(err)
    }
}
```

## Streaming vs Pagination Comparison

| | Streaming | Pagination |
|--|-----------|-----------|
| Memory | O(1) - constant | O(page size) |
| Access pattern | sequential only | random page access |
| Best for | ETL, aggregation, export | display, UI paging |
| Total count | not available | available with $count |
| Interruptible | yes, via Close() | yes, stop calling NextPage |

!!! warning "Sequential access only"
    Streams are forward-only. If you need to re-process earlier items, collect them into a slice first (accepting the memory cost) or use pagination.

## Related Pages

- [Typed Pagination](pagination.md) - Page-by-page access with random navigation
- [Delta Sync](delta-sync.md) - Stream only changed records since last sync
- [Query Builder](query-builder.md) - Building the initial query
