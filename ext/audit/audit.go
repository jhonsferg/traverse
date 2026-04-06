// Package audit provides an OData audit trail extension for the traverse client.
// It intercepts all HTTP requests made by the traverse relay transport and records
// an AuditEntry for each operation, capturing the operation type, entity set,
// entity key, duration, status code, user ID, request ID, and any error.
package audit

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jhonsferg/relay"
)

// OperationType is the type of OData operation performed.
type OperationType string

const (
	OperationRead   OperationType = "READ"
	OperationCreate OperationType = "CREATE"
	OperationUpdate OperationType = "UPDATE"
	OperationDelete OperationType = "DELETE"
	OperationBatch  OperationType = "BATCH"
)

// AuditEntry records a single OData operation.
type AuditEntry struct {
	Timestamp  time.Time
	Operation  OperationType
	EntitySet  string
	EntityKey  string // may be empty for collection operations
	URL        string // full URL of the request
	StatusCode int
	Duration   time.Duration
	UserID     string // from context via AuditUserKey
	RequestID  string // from context via AuditRequestIDKey
	Error      string // non-empty if operation failed
	Metadata   map[string]string
}

// AuditLogger receives audit entries.
type AuditLogger interface {
	Log(ctx context.Context, entry AuditEntry)
}

// AuditLoggerFunc adapts a function to AuditLogger.
type AuditLoggerFunc func(ctx context.Context, entry AuditEntry)

// Log calls f(ctx, entry).
func (f AuditLoggerFunc) Log(ctx context.Context, entry AuditEntry) { f(ctx, entry) }

// contextKey is the unexported type for context keys in this package.
type contextKey string

const (
	// AuditUserKey is the context key for the user ID included in audit entries.
	AuditUserKey contextKey = "audit.user_id"
	// AuditRequestIDKey is the context key for the request ID included in audit entries.
	AuditRequestIDKey contextKey = "audit.request_id"
)

// WithUser returns a new context that carries the given user ID.
// The value is picked up by the audit transport middleware and recorded in AuditEntry.UserID.
func WithUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, AuditUserKey, userID)
}

// WithRequestID returns a new context that carries the given request ID.
// The value is picked up by the audit transport middleware and recorded in AuditEntry.RequestID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, AuditRequestIDKey, requestID)
}

// entityKeyRe matches an OData entity key expression, e.g. Orders(42) or Products('ABC').
var entityKeyRe = regexp.MustCompile(`^([^(]+)\(([^)]*)\)$`)

// parseEntityInfo extracts the entity set name and key from the last non-empty
// path segment of the given URL path.
//
//   - /odata/Orders       → ("Orders", "")
//   - /odata/Orders(1)    → ("Orders", "1")
//   - /odata/$batch       → ("$batch", "")
func parseEntityInfo(urlPath string) (entitySet, entityKey string) {
	// Strip trailing slash and query string (caller passes path only)
	path := strings.TrimRight(urlPath, "/")
	// Find the last segment
	idx := strings.LastIndex(path, "/")
	segment := path
	if idx >= 0 {
		segment = path[idx+1:]
	}
	if segment == "" {
		return "", ""
	}
	if m := entityKeyRe.FindStringSubmatch(segment); m != nil {
		return m[1], m[2]
	}
	return segment, ""
}

// operationFromRequest derives the OData OperationType from the HTTP method
// and the entity set name.
func operationFromRequest(method, entitySet string) OperationType {
	switch strings.ToUpper(method) {
	case http.MethodGet:
		return OperationRead
	case http.MethodPost:
		if entitySet == "$batch" {
			return OperationBatch
		}
		return OperationCreate
	case http.MethodPatch, http.MethodPut:
		return OperationUpdate
	case http.MethodDelete:
		return OperationDelete
	default:
		return OperationRead
	}
}

// WithAuditTrail returns a relay.Option that wraps the HTTP transport with an
// audit middleware. The middleware intercepts every request, records the
// elapsed time and response details, and calls logger.Log with an AuditEntry.
//
// Usage:
//
//	logger := audit.AuditLoggerFunc(func(ctx context.Context, e audit.AuditEntry) {
//	    log.Printf("[AUDIT] %s %s %d %s", e.Operation, e.EntitySet, e.StatusCode, e.Duration)
//	})
//	client, err := traverse.New("https://api.example.com/odata/",
//	    traverse.WithHTTPOption(audit.WithAuditTrail(logger)),
//	)
func WithAuditTrail(logger AuditLogger) relay.Option {
	return relay.WithTransportMiddleware(Middleware(logger))
}

// Middleware returns a transport middleware function suitable for direct use with
// relay.WithTransportMiddleware. It is exposed separately for callers that build
// their own relay.Client or http.Client.
func Middleware(logger AuditLogger) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			ctx := req.Context()
			start := time.Now()

			entitySet, entityKey := parseEntityInfo(req.URL.Path)
			op := operationFromRequest(req.Method, entitySet)

			userID, _ := ctx.Value(AuditUserKey).(string)
			requestID, _ := ctx.Value(AuditRequestIDKey).(string)

			resp, err := next.RoundTrip(req)

			entry := AuditEntry{
				Timestamp: start,
				Operation: op,
				EntitySet: entitySet,
				EntityKey: entityKey,
				URL:       req.URL.String(),
				Duration:  time.Since(start),
				UserID:    userID,
				RequestID: requestID,
			}

			if err != nil {
				entry.Error = err.Error()
			} else if resp != nil {
				entry.StatusCode = resp.StatusCode
				if resp.StatusCode >= 400 {
					entry.Error = resp.Status
				}
			}

			logger.Log(ctx, entry)
			return resp, err
		})
	}
}

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// InMemoryAuditLog is a thread-safe in-memory AuditLogger useful for testing.
type InMemoryAuditLog struct {
	mu      sync.Mutex
	entries []AuditEntry
}

// NewInMemoryAuditLog creates a new, empty InMemoryAuditLog.
func NewInMemoryAuditLog() *InMemoryAuditLog {
	return &InMemoryAuditLog{}
}

// Log appends entry to the in-memory log.
func (l *InMemoryAuditLog) Log(_ context.Context, entry AuditEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, entry)
}

// Entries returns a snapshot of all recorded audit entries.
func (l *InMemoryAuditLog) Entries() []AuditEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]AuditEntry, len(l.entries))
	copy(out, l.entries)
	return out
}

// Clear removes all recorded audit entries.
func (l *InMemoryAuditLog) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = l.entries[:0]
}
