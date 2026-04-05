# Async Operations

Some OData services - particularly SAP - respond to long-running operations with `202 Accepted` and a `Location` header pointing to a status endpoint. traverse's `AsyncOpPoller` handles this automatically.

## The 202 + Location Pattern

```
POST /LargeReportGenerate
< 202 Accepted
< Location: /AsyncOperationStatus('abc-123')
```

The client polls the Location URL until the operation completes:

```
GET /AsyncOperationStatus('abc-123')
< 200 OK  {"Status": "Running"}

GET /AsyncOperationStatus('abc-123')
< 200 OK  {"Status": "Running"}

GET /AsyncOperationStatus('abc-123')
< 200 OK  {"Status": "Succeeded", "ResultURL": "..."}
```

## AsyncOpPoller

```go
poller, err := traverse.NewAsyncPoller(client, operationURL)
if err != nil {
    log.Fatal(err)
}

status, err := poller.Wait(ctx)
if err != nil {
    switch {
    case errors.Is(err, traverse.ErrAsyncOpFailed):
        fmt.Println("operation failed on server")
    case errors.Is(err, traverse.ErrAsyncOpTimeout):
        fmt.Println("operation timed out (max polls reached)")
    default:
        log.Fatal(err)
    }
}
fmt.Printf("Completed with status: %s\n", status)
```

## Creating a Poller from an Operation

When you call an OData action that may return 202:

```go
result, err := client.Action("GenerateReport").
    Parameter("Year", 2024).
    Execute(ctx)

if result.IsAsync() {
    poller, err := traverse.NewAsyncPoller(client, result.Location())
    if err != nil {
        log.Fatal(err)
    }
    status, err := poller.Wait(ctx)
    // ...
}
```

## Configuring the Poller

### WithPollInterval

Set how often to poll the status endpoint:

```go
poller := traverse.NewAsyncPoller(client, url,
    traverse.WithPollInterval(5 * time.Second),
)
```

Default: 2 seconds.

### WithMaxPolls

Set the maximum number of polling attempts before returning `ErrAsyncOpTimeout`:

```go
poller := traverse.NewAsyncPoller(client, url,
    traverse.WithMaxPolls(60), // give up after 60 polls
)
```

Default: 30 polls (so 60 seconds at default interval).

## AsyncOpStatus Constants

| Constant | Meaning |
|----------|---------|
| `AsyncOpRunning` | Operation is still in progress |
| `AsyncOpSucceeded` | Operation completed successfully |
| `AsyncOpFailed` | Operation completed with an error |
| `AsyncOpCancelled` | Operation was cancelled |

## Complete Example

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "time"

    "github.com/jhonsferg/traverse"
)

func runLongReport(ctx context.Context, client *traverse.Client, year int) error {
    // Trigger the async operation
    result, err := client.Action("SalesReportGenerate").
        Parameter("Year", year).
        Execute(ctx)
    if err != nil {
        return fmt.Errorf("trigger: %w", err)
    }

    if !result.IsAsync() {
        // Completed synchronously
        fmt.Println("Report generated synchronously")
        return nil
    }

    fmt.Printf("Async operation started: %s\n", result.Location())

    poller, err := traverse.NewAsyncPoller(client, result.Location(),
        traverse.WithPollInterval(3*time.Second),
        traverse.WithMaxPolls(100),
    )
    if err != nil {
        return err
    }

    fmt.Print("Waiting")
    done := make(chan struct{})
    go func() {
        defer close(done)
        for {
            select {
            case <-ctx.Done():
                return
            case <-time.After(500 * time.Millisecond):
                fmt.Print(".")
            }
        }
    }()

    status, err := poller.Wait(ctx)
    close(done)
    fmt.Println()

    if err != nil {
        if errors.Is(err, traverse.ErrAsyncOpFailed) {
            return fmt.Errorf("report generation failed on server")
        }
        if errors.Is(err, traverse.ErrAsyncOpTimeout) {
            return fmt.Errorf("timed out after 300s")
        }
        return err
    }

    fmt.Printf("Report ready: status=%s\n", status)
    return nil
}
```

## SAP Long-Running Operations

SAP ABAP Gateway uses this pattern for operations like:
- Large data exports
- Report generation
- Batch posting (FB01, etc.)
- Background job submission

The `Location` header in SAP responses typically points to a polling entity like `/sap/opu/odata/sap/MY_SRV/OperationStatusSet('handle-123')`.

!!! tip "Context cancellation"
    Pass a context with a timeout to bound the total wait time regardless of `WithMaxPolls`:

    ```go
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()
    status, err := poller.Wait(ctx)
    ```

## Related Pages

- [Async Poller Reference](../reference/async-op.md) - Full API reference
- [SAP Compatibility](sap.md) - SAP-specific async patterns
