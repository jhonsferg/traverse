# The Name

## Why *traverse*?

The verb *to traverse* means to walk through something large, complex, or extended  -  one step at a time, without needing to hold the whole thing in memory at once. In computer science, *tree traversal* and *graph traversal* describe exactly that: visiting every node of a structure incrementally, following pointers from one element to the next, rather than materialising the entire structure before you begin.

That is the problem this library was built to solve.

When real production workloads against SAP S/4HANA were profiled, the pattern was the same across every team that had attempted it before: existing Go OData clients would request a collection, accumulate the full response in memory, and either exhaust the heap or hit a timeout before returning a single record to the caller. A service with one million materials became effectively unreachable  -  not because the network was slow or SAP was down, but because the client's model was wrong.

```
# other clients
GET /MaterialSet → accumulate 1 000 000 records in RAM → out of memory

# traverse
GET /MaterialSet → visit each record one by one → constant memory
```

The difference is not *what* you fetch. It is *how* you move through it.

---

## Three principles the name implies

Naming the library *traverse* was not arbitrary. The word imposes a discipline, and the implementation honours it.

### The path matters more than the destination

In a traversal, you do not wait until you have everything before you start working. Each node is actionable the moment you reach it. Traverse expresses this directly in its API:

```go
for result := range client.From("MaterialSet").Stream(ctx) {
    if result.Err != nil {
        // handle error and continue  -  no data is lost
        continue
    }
    process(result.Value) // available immediately, not after page N is done
}
```

The caller's goroutine and the HTTP fetcher run concurrently. Page prefetching means the next page is already in flight while you process the current one. The result is a pipeline, not a batch.

### Respect for the terrain

A careful traversal does not tear up the ground beneath it. When you traverse a forest you follow the paths; you do not bulldoze trees to make the route shorter.

Traverse is deliberately gentle on the services it talks to:

- **Rate limiting and circuit breaking** are inherited from [relay](https://github.com/jhonsferg/relay), the HTTP transport layer beneath traverse. If SAP is under load, traverse backs off  -  it does not retry in a tight loop.
- **Pagination follows the server's rhythm.** Traverse reads `@odata.nextLink` and `$skiptoken` and follows them. It does not inject artificial `$top`/`$skip` values that the server might not honour efficiently.
- **CSRF tokens are managed transparently.** SAP OData v2 write operations require a valid `X-CSRF-Token`. Traverse (via `ext/sap`) fetches and caches the token, invalidates it on 403, and retries  -  without the caller ever touching the handshake.

### The map is not the territory

A tree traversal algorithm does not require the full tree in memory  -  only the current node, a reference to its children, and a stack or queue to track where to go next. The algorithm is O(1) in working memory regardless of the tree's depth or breadth.

Traverse applies the same principle to remote data:

- You can stream one million SAP sales orders without keeping one million structs alive simultaneously.
- The `Paginator[T]` type holds exactly one page at a time; when you advance past the last element, the previous page is eligible for garbage collection.
- Metadata (the OData `$metadata` EDMX document) is cached once and reused, so schema knowledge is cheap to acquire and free to hold.

---

## What the name does not promise

Being honest about the name's limitations matters.

*traverse* does not evoke OData or SAP. Someone searching for a "golang odata client" will not instinctively land on *traverse* without context. Libraries with explicit OData or SAP in their names have better organic discoverability.

The deliberate trade-off: protocol-specific names age poorly. If *traverse* tomorrow supports GraphQL cursor-based pagination, or any other collection protocol built on the same incremental-fetch principle, the name remains accurate. *go-odata* would not.

---

## Names that were not chosen

During the design phase, several names were evaluated:

| Name | Why it was rejected |
|---|---|
| `odata-go` | Descriptive but generic; no character; hard to distinguish in search results |
| `sapient` | Wordplay on SAP + *sapient* (wise), but implies SAP-only when OData is protocol-agnostic |
| `flow` | Captures streaming but too generic; namespace collisions in the Go ecosystem |
| `cursor` | Technically precise  -  a database cursor is exactly this  -  but connotes SQL over HTTP/OData |
| `scroll` | Evokes incremental retrieval well, but slightly informal for a production library |

*traverse* was chosen because it is an active verb, has no known collisions in the Go OData library space, and because the metaphor is honest  -  it does not promise more than the library delivers.

---

## Summary

The name *traverse* is not a backronym. It is not a brand constructed after the fact. It is the most accurate English verb for what the library does with a million records: it walks through them, one at a time, without ever needing to hold the whole collection in memory.

That constraint  -  *memory proportional to a page, not to the dataset*  -  is the founding invariant of the library. Everything else follows from it.
