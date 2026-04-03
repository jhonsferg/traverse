// Package traverse provides a production-grade OData v2/v4 client for Go,
// built on top of github.com/jhonsferg/relay.
//
// traverse is designed to consume OData services with millions of records
// without running out of memory, using server-side pagination, streaming
// JSON decoding, and Go channels for backpressure.
//
// It is compatible with SAP OData services (both classic ABAP Gateway
// using OData v2 and S/4HANA using OData v4).
//
// Quick start:
//
//	client, err := traverse.New(
//	    traverse.WithBaseURL("https://sap.example.com/sap/opu/odata/sap/MY_SRV"),
//	    traverse.WithBasicAuth("user", "pass"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	for result := range client.From("MaterialSet").Stream(ctx) {
//	    if result.Err != nil {
//	        log.Fatal(result.Err)
//	    }
//	    fmt.Println(result.Value) // map[string]any for each record
//	}
//
// See https://github.com/jhonsferg/traverse for full documentation.
package traverse
