# traverse-tui — Interactive CLI

`traverse-tui` is an interactive terminal application for building and executing OData queries against any OData v4 service, without writing code.

## Installation

```bash
go install github.com/jhonsferg/traverse/cmd/traverse-tui@latest
```

## Starting the CLI

```bash
traverse-tui --base-url https://services.odata.org/V4/Northwind/Northwind.svc
```

If `--base-url` is omitted, set it interactively with the `url` command.

## Commands

| Command | Aliases | Syntax | Description |
|---------|---------|--------|-------------|
| `url` | | `url <base-url>` | Set or change the OData service base URL |
| `use` | | `use <EntitySet>` | Select the entity set to query |
| `filter` | `where` | `filter <expression>` | Append a `$filter` clause (cumulative) |
| `select` | | `select <f1,f2,...>` | Set `$select` fields (comma-separated) |
| `orderby` | | `orderby <field [asc\|desc]>` | Set `$orderby` |
| `top` | | `top <n>` | Set `$top` (default: 10) |
| `skip` | | `skip <n>` | Set `$skip` offset |
| `show` | `query` | `show` | Print the full OData URL without executing |
| `run` | `exec`, `go` | `run` | Execute the query and display results |
| `status` | | `status` | Show all current query parameters |
| `reset` | `clear` | `reset` | Clear all query parameters (keeps entity set and base URL) |
| `help` | `h`, `?` | `help` | List available commands |
| `exit` | `quit`, `q` | `exit` | Quit the CLI |

## Interactive session example

```
$ traverse-tui --base-url https://services.odata.org/V4/Northwind/Northwind.svc

 ████████╗██████╗  █████╗ ██╗   ██╗███████╗██████╗ ███████╗███████╗
 ...
                     OData Interactive Query Builder

Type 'help' for available commands, 'exit' to quit.

traverse[<no entity set>]> use Customers
traverse[Customers]> filter Country eq 'Germany'
Filter added: Country eq 'Germany'

traverse[Customers]> select CustomerID,CompanyName,Country
Select: CustomerID,CompanyName,Country

traverse[Customers]> top 5
Top: 5

traverse[Customers]> show
https://services.odata.org/V4/Northwind/Northwind.svc/Customers?$filter=Country+eq+%27Germany%27&$select=CustomerID%2CCompanyName%2CCountry&$top=5

traverse[Customers]> run
{"@odata.context":"...","value":[
  {"CustomerID":"ALFKI","CompanyName":"Alfreds Futterkiste","Country":"Germany"},
  ...
]}

traverse[Customers]> status
Base URL  : https://services.odata.org/V4/Northwind/Northwind.svc
Entity Set: Customers
Filter    : [Country eq 'Germany']
Select    : [CustomerID CompanyName Country]
Top       : 5
Skip      : 0

traverse[Customers]> reset
traverse[Customers]> exit

Goodbye!
```

## Notes / Limitations

- Multiple `filter` calls are cumulative; each adds another clause joined with `and` in the URL.
- `orderby` accepts only a single expression; chain multiple fields manually (e.g. `orderby LastName asc,FirstName asc`).
- Results are printed as raw JSON. Pretty-printing requires piping to a tool like `jq`.
- The CLI does not support authentication; for protected endpoints wrap the service behind a proxy or use a pre-shared token in the base URL.
- `traverse-tui` uses OData v4 query syntax; v2 services may behave differently.
