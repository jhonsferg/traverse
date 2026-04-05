# Microsoft Graph API Adapter

traverse includes a first-class adapter for the **Microsoft Graph API**, which is itself built on OData v4. The adapter pre-configures authentication, the correct base URL, and Graph-specific headers so you can start querying immediately.

## Setup

```go
import "github.com/jhonsferg/traverse"

graphClient := traverse.NewGraphClient(traverse.GraphConfig{
    AccessToken: "eyJ0eXAi...",  // Azure AD / MSAL access token
})
```

This creates a traverse client pre-configured with:

- Base URL: `https://graph.microsoft.com/v1.0`
- `Authorization: Bearer <token>` header
- OData v4 protocol
- Graph error decoding

## Configuration Reference

```go
type GraphConfig struct {
    // API version: "v1.0" (default) or "beta"
    Version string

    // Bearer token from Azure AD / MSAL
    AccessToken string

    // OData ConsistencyLevel header - required for $count queries
    // Set to "eventual" when using $count
    ConsistencyLevel string
}
```

## Querying Users

```go
type User struct {
    ID          string `json:"id"`
    DisplayName string `json:"displayName"`
    Mail        string `json:"mail"`
    JobTitle    string `json:"jobTitle"`
}

gc := traverse.NewGraphClient(traverse.GraphConfig{
    AccessToken: token,
})

users := traverse.From[User](gc, "users")

// Get all users in Marketing department
results, err := users.
    Filter("department eq 'Marketing'").
    Select("id", "displayName", "mail", "jobTitle").
    OrderBy("displayName").
    Top(50).
    List(ctx)
```

## Working with Groups

```go
type Group struct {
    ID          string   `json:"id"`
    DisplayName string   `json:"displayName"`
    Mail        string   `json:"mail"`
    GroupTypes  []string `json:"groupTypes"`
}

groups := traverse.From[Group](gc, "groups")

// Get Microsoft 365 groups only
m365Groups, err := groups.
    Filter("groupTypes/any(c:c eq 'Unified')").
    Select("id", "displayName", "mail").
    List(ctx)
```

## Accessing the Beta API

```go
gcBeta := traverse.NewGraphClient(traverse.GraphConfig{
    AccessToken: token,
    Version:     "beta",
})

type EmployeeExperience struct {
    ID string `json:"id"`
}

// Beta endpoint
result, err := traverse.From[EmployeeExperience](gcBeta, "employeeExperience/learningProviders").
    List(ctx)
```

## Using $count

To use `$count`, set `ConsistencyLevel: "eventual"`:

```go
gc := traverse.NewGraphClient(traverse.GraphConfig{
    AccessToken:      token,
    ConsistencyLevel: "eventual",
})

users := traverse.From[User](gc, "users")

// $count requires ConsistencyLevel: eventual
countResult, err := users.
    Filter("accountEnabled eq true").
    Count(ctx)
```

## Navigating Relationships

```go
type Message struct {
    ID      string `json:"id"`
    Subject string `json:"subject"`
    From    struct {
        EmailAddress struct {
            Address string `json:"address"`
        } `json:"emailAddress"`
    } `json:"from"`
}

// Get messages from a user's inbox
messages := traverse.From[Message](gc, "users/me/mailFolders/inbox/messages")

inbox, err := messages.
    Select("id", "subject", "from").
    OrderBy("receivedDateTime desc").
    Top(25).
    List(ctx)
```

## Error Handling

Graph API errors are decoded into a structured `GraphError`:

```go
type GraphError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

```go
_, err := users.Key("nonexistent-id").Get(ctx)
var ge *traverse.GraphError
if errors.As(err, &ge) {
    fmt.Printf("Graph error %s: %s\n", ge.Code, ge.Message)
    // e.g. "Request_ResourceNotFound: Resource 'nonexistent-id' does not exist"
}
```

## Complete Example: Org Chart

```go
package main

import (
    "context"
    "fmt"
    "log"
    "github.com/jhonsferg/traverse"
)

type Employee struct {
    ID          string `json:"id"`
    DisplayName string `json:"displayName"`
    JobTitle    string `json:"jobTitle"`
    Department  string `json:"department"`
    ManagerID   string `json:"managerId,omitempty"`
}

func main() {
    gc := traverse.NewGraphClient(traverse.GraphConfig{
        AccessToken: getToken(), // your token acquisition function
    })

    employees := traverse.From[Employee](gc, "users")

    // Get engineering department
    eng, err := employees.
        Filter("department eq 'Engineering'").
        Select("id", "displayName", "jobTitle").
        OrderBy("displayName").
        List(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    for _, e := range eng {
        fmt.Printf("%s - %s\n", e.DisplayName, e.JobTitle)
    }
}
```
