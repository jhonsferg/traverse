# Async Poller Reference

`AsyncPoller[T]` manages the poll loop for OData long-running operations that return `202 Accepted` with a `Location` header.

## Type definition

```go
type AsyncPoller[T any] struct { /* unexported */ }

// Poll sends one poll request to the operation status URL.
// Returns (true, nil) when the operation is still running.
// Returns (false, nil) when the operation completed successfully.
// Returns (false, err) on failure or cancellation.
func (p *AsyncPoller[T]) Poll(ctx context.Context) (running bool, err error)

// Result returns the final typed result after polling completes.
// Must only be called after Poll returns (false, nil).
func (p *AsyncPoller[T]) Result() (T, error)

// Wait polls until completion, context cancellation, or MaxWait.
func (p *AsyncPoller[T]) Wait(ctx context.Context) (T, error)

// Status returns the last seen operation status string.
func (p *AsyncPoller[T]) Status() string

// PercentComplete returns the last seen completion percentage (0-100).
// Returns -1 if the server did not report progress.
func (p *AsyncPoller[T]) PercentComplete() int
```

## AsyncOptions

```go
type AsyncOptions struct {
    // PollInterval is the time between poll requests. Default: 2s.
    PollInterval time.Duration

    // MaxWait is the maximum total time to wait. Default: 10 minutes.
    MaxWait time.Duration

    // OnProgress is called after each poll with the current status.
    OnProgress func(status string, pct int)
}
```

## Creating a poller

```go
// Trigger an async operation
poller, err := client.Collection("ExportJobs").
    CreateAsync[ExportResult](ctx, exportRequest, traverse.AsyncOptions{
        PollInterval: 5 * time.Second,
        MaxWait:      10 * time.Minute,
    })
```

## Manual poll loop

```go
for {
    running, err := poller.Poll(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if !running {
        break
    }
    log.Printf("status: %s (%d%%)", poller.Status(), poller.PercentComplete())
    time.Sleep(3 * time.Second)
}

result, err := poller.Result()
```

## Wait for completion

```go
result, err := poller.Wait(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println("export URL:", result.DownloadURL)
```

## See also

- [Async Operations guide](../guides/async-operations.md)
- [Client Reference](client.md)
