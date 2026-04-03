package traverse

import (
	"encoding/json"
	"fmt"
	"io"
)

// parseODataV4ServiceDocument parses an OData v4 service document.
//
// parseODataV4ServiceDocument decodes a JSON response from the OData v4 service root,
// which lists all available entity sets in the service.
//
// OData v4 service documents follow this format:
//
//	{
//	  "value": [
//	    {"name": "EntitySet1", "url": "EntitySet1"},
//	    {"name": "EntitySet2", "url": "EntitySet2"}
//	  ]
//	}
//
// Returns a [ServiceDocument] with the parsed entity sets, or an error if decoding fails.
func parseODataV4ServiceDocument(reader io.Reader) (*ServiceDocument, error) {
	var wrapper struct {
		Value []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"value"`
	}

	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode OData v4 service document: %w", err)
	}

	doc := &ServiceDocument{
		EntitySets: make([]EntitySetReference, 0, len(wrapper.Value)),
	}

	for _, es := range wrapper.Value {
		doc.EntitySets = append(doc.EntitySets, EntitySetReference{
			Name: es.Name,
			URL:  es.URL,
		})
	}

	return doc, nil
}

// parseODataV2ServiceDocument parses an OData v2 service document.
//
// parseODataV2ServiceDocument decodes a JSON response from the OData v2 service root,
// which lists all available entity sets in the service.
//
// OData v2 service documents are wrapped in "d" and may use either "EntitySets"
// or "results" to list entity sets (SAP systems sometimes use "results"):
//
//	{
//	  "d": {
//	    "EntitySets": [
//	      {"__metadata": {...}, "Name": "EntitySet1", "Url": "EntitySet1"},
//	      {"Name": "EntitySet2", "Url": "EntitySet2"}
//	    ]
//	  }
//	}
//
// OR (alternate format):
//
//	{
//	  "d": {
//	    "results": [
//	      {"Name": "EntitySet1", "Url": "EntitySet1"},
//	      {"Name": "EntitySet2", "Url": "EntitySet2"}
//	    ]
//	  }
//	}
//
// This function tries EntitySets first, then falls back to results format.
// Returns a [ServiceDocument] with the parsed entity sets, or an error if decoding fails.
func parseODataV2ServiceDocument(reader io.Reader) (*ServiceDocument, error) {
	var wrapper struct {
		D struct {
			// Try both possible field names
			EntitySets []struct {
				Name string `json:"Name"`
				URL  string `json:"Url"`
			} `json:"EntitySets"`
			Results []struct {
				Name string `json:"Name"`
				URL  string `json:"Url"`
			} `json:"results"`
		} `json:"d"`
	}

	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode OData v2 service document: %w", err)
	}

	doc := &ServiceDocument{
		EntitySets: make([]EntitySetReference, 0),
	}

	// Try EntitySets first (primary format)
	if len(wrapper.D.EntitySets) > 0 {
		for _, es := range wrapper.D.EntitySets {
			doc.EntitySets = append(doc.EntitySets, EntitySetReference{
				Name: es.Name,
				URL:  es.URL,
			})
		}
	} else if len(wrapper.D.Results) > 0 {
		// Fallback to results format
		for _, es := range wrapper.D.Results {
			doc.EntitySets = append(doc.EntitySets, EntitySetReference{
				Name: es.Name,
				URL:  es.URL,
			})
		}
	}

	return doc, nil
}
