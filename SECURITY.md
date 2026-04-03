# Security Policy

## Reporting Security Vulnerabilities

**Do not file public issues for security vulnerabilities.**

If you discover a security vulnerability in Traverse, please email [security@example.com](mailto:security@example.com) with:

1. Description of the vulnerability
2. Steps to reproduce (if possible)
3. Potential impact
4. Your contact information

We will:
- Acknowledge your report within 24 hours
- Assess the severity
- Develop and test a fix
- Release a patched version
- Credit you in the release notes (with permission)

## Security Considerations

### For Users

1. **Always use HTTPS**: Never send credentials over HTTP
2. **Validate certificates**: Verify SSL/TLS certificates in production
3. **Protect credentials**: Store auth credentials securely (environment variables, vaults)
4. **Update regularly**: Keep Traverse updated for security patches
5. **Monitor logs**: Watch for unusual OData query patterns

### For Contributors

When contributing, be mindful of:

1. **Input validation**: Always validate and sanitize OData filter expressions
2. **Credential handling**: Never log passwords, tokens, or sensitive data
3. **CSRF protection**: Always require CSRF tokens for write operations
4. **SQL injection prevention**: Avoid constructing filters from untrusted input
5. **Dependency updates**: Keep dependencies up-to-date

## Known Security Limitations

- CSRF token caching TTL is 30 minutes by default (configurable)
- Bearer token authentication requires custom implementation
- Client-level authentication middleware integration pending relay updates

## Best Practices

### SAP Integration

```go
// ✗ Bad: Credentials in code
client, _ := sap.NewSAPClient(
	sap.WithSystemURL("http://sap.example.com"),
	sap.WithBasicAuth("username", "password123"),
)

// ✓ Good: Credentials from environment
username := os.Getenv("SAP_USERNAME")
password := os.Getenv("SAP_PASSWORD")
client, _ := sap.NewSAPClient(
	sap.WithSystemURL(os.Getenv("SAP_URL")),
	sap.WithBasicAuth(username, password),
)
```

### Query Construction

```go
// ✗ Bad: Unsanitized user input in filter
userInput := getUserInput()  // "Name eq 'test' or 1=1"
qb := client.From("Users").Filter(userInput)

// ✓ Good: Parameterized filters
qb := client.From("Users").
	Where("Name").Eq(userInput)  // Properly escaped
```

### Error Handling

```go
// ✗ Bad: Exposing sensitive info in logs
log.Printf("Auth failed: %v\n", err)  // Might include token

// ✓ Good: Safe error logging
log.Printf("Auth failed: %v\n", "Invalid credentials")
```

## Supported Versions

| Version | Status | Support Until |
|---------|--------|---------------|
| 1.x     | Future | TBD           |
| 0.x     | Current| Until 1.0     |

Security patches will be released for:
- Current version (0.x)
- Previous major version (if applicable)

## Compliance

Traverse aims to comply with:
- OWASP Top 10
- CWE Top 25
- SAP security guidelines
- Industry best practices

## Automated Security Scanning

Traverse implements multiple layers of security scanning in its CI/CD pipeline:

### 1. **Trivy Vulnerability Scanning** 🔍
- Scans for known CVEs in Go modules
- Runs on every push and pull request
- Blocks PRs with CRITICAL vulnerabilities
- Generates SARIF reports in GitHub Security tab
- **Trigger**: `.github/workflows/trivy.yml`

### 2. **TruffleHog Secrets Scanning** 🔐
- Detects exposed credentials, API keys, tokens
- Scans both filesystem and git history
- Blocks PRs if secrets are detected
- Supports encrypted and obfuscated secrets
- **Trigger**: `.github/workflows/secrets-scan.yml`

### 3. **Nancy Supply Chain Security** 📊
- Detects malicious packages in dependencies
- Identifies vulnerable Go modules
- Uses Sonatype OSS Index database (real-time)
- Prevents typosquatting attacks
- **Trigger**: `.github/workflows/nancy.yml`

### 4. **CodeQL Static Analysis** 🛡️
- Semantic code analysis for security issues
- Runs on every commit (scheduled + event-driven)
- Detects: SQLi, XSS, SSRF, race conditions
- **Trigger**: `.github/workflows/codeql.yml`

### 5. **Dependabot Automated Updates** 🔄
- Automatically checks for dependency updates
- Creates PRs for security updates immediately
- Groups updates by type and severity
- **Configuration**: `.github/dependabot.yml`

### 6. **SBOM Generation & cosign Signing** 📋
- Generates Software Bill of Materials (SPDX + CycloneDX)
- Cryptographically signs all release artifacts
- Enables supply chain verification
- **Trigger**: `.github/workflows/sbom-sign.yml`

### Continuous Dependency Audits

Manual audit commands available:

```bash
# Check Go modules for vulnerabilities
go list -json -m all | nancy sleuth

# Verify module integrity
go mod verify

# Check for available updates
go list -u -m all

# Run Trivy scan locally
trivy config .

# Scan for secrets
trufflehog filesystem .
```

### Dependencies

This project depends on:
- `github.com/jhonsferg/relay` - HTTP client library

All dependencies are:
- Automatically scanned for vulnerabilities
- Kept updated via Dependabot
- Included in SBOM for transparency

## Responsible Disclosure Timeline

1. **Day 0**: Vulnerability report received
2. **Day 1**: Acknowledgment and severity assessment
3. **Day 7**: Security patch prepared
4. **Day 14**: Patch released publicly
5. **Day 21**: Security advisory published

(Timeline may vary based on severity)

## Questions?

For security-related questions (non-disclosure):
- Email: [security@example.com](mailto:security@example.com)
- Discussions: [GitHub Discussions](https://github.com/jhonsferg/traverse/discussions)

---

**Thank you for helping keep Traverse secure!** 🔒
