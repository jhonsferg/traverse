package traverse

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// atomFeedParser implements token-by-token streaming of an OData v2 Atom/XML feed.
//
// The Atom format for OData v2 uses the following structure:
//
//	<feed xmlns="http://www.w3.org/2005/Atom"
//	      xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
//	      xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
//	  <m:count>100</m:count>
//	  <link rel="next" href="..."/>
//	  <entry>
//	    <id>...</id>
//	    <content type="application/xml">
//	      <m:properties>
//	        <d:OrderID m:type="Edm.Int32">1</d:OrderID>
//	        <d:Status>Active</d:Status>
//	      </m:properties>
//	    </content>
//	  </entry>
//	</feed>
//
// ParseAtomFeed reads directly from the io.Reader and populates the Page.
func ParseAtomFeed(r io.Reader, page *Page) error {
	dec := xml.NewDecoder(r)

	const (
		atomNS     = "http://www.w3.org/2005/Atom"
		dataServNS = "http://schemas.microsoft.com/ado/2007/08/dataservices"
		metaNS     = "http://schemas.microsoft.com/ado/2007/08/dataservices/metadata"
	)

	var (
		inEntry      bool
		inProperties bool
		inCount      bool
		currentProp  string
		currentEntry map[string]interface{}
	)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("traverse: atom feed parse error: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case t.Name.Space == atomNS && t.Name.Local == "feed":
				// root element, nothing to do

			case t.Name.Space == atomNS && t.Name.Local == "link":
				// Check for next link: <link rel="next" href="..."/>
				var rel, href string
				for _, attr := range t.Attr {
					switch attr.Name.Local {
					case "rel":
						rel = attr.Value
					case "href":
						href = attr.Value
					}
				}
				if rel == "next" && href != "" {
					page.NextLink = href
				}

			case t.Name.Space == metaNS && t.Name.Local == "count":
				inCount = true

			case t.Name.Space == atomNS && t.Name.Local == "entry":
				inEntry = true
				currentEntry = make(map[string]interface{}, 16)

			case inEntry && t.Name.Space == metaNS && t.Name.Local == "properties":
				inProperties = true

			case inProperties && t.Name.Space == dataServNS:
				// <d:FieldName m:type="..."> - record the field name for CharData
				currentProp = t.Name.Local
				// Check for null indicator
				for _, attr := range t.Attr {
					if attr.Name.Space == metaNS && attr.Name.Local == "null" && attr.Value == "true" {
						currentEntry[currentProp] = nil
						currentProp = "" // suppress CharData collection
					}
				}
			}

		case xml.EndElement:
			switch {
			case t.Name.Space == atomNS && t.Name.Local == "entry":
				if inEntry && currentEntry != nil {
					page.Value = append(page.Value, currentEntry)
				}
				inEntry = false
				currentEntry = nil
				inProperties = false
				currentProp = ""

			case inEntry && t.Name.Space == metaNS && t.Name.Local == "properties":
				inProperties = false

			case inProperties && t.Name.Space == dataServNS:
				currentProp = ""

			case t.Name.Space == metaNS && t.Name.Local == "count":
				inCount = false
			}

		case xml.CharData:
			val := strings.TrimSpace(string(t))
			if val == "" {
				continue
			}
			switch {
			case inCount:
				// Parse the count value
				parseAtomCount(val, page)

			case inProperties && currentProp != "" && currentEntry != nil:
				// Assign field value; keep as string (JSON decoding does the same)
				currentEntry[currentProp] = val
			}
		}
	}

	return nil
}

// parseAtomCount parses the <m:count> text value and sets Page.Count.
func parseAtomCount(val string, page *Page) {
	val = strings.TrimSpace(val)
	if val == "" {
		return
	}
	// Try integer first, then float (OData v2 sometimes sends float)
	var n int64
	if _, err := fmt.Sscanf(val, "%d", &n); err == nil {
		page.Count = &n
		return
	}
	var f float64
	if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
		n = int64(f)
		page.Count = &n
	}
}

// IsAtomContentType returns true when the Content-Type header indicates an Atom/XML feed.
func IsAtomContentType(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "application/atom") ||
		strings.Contains(ct, "application/xml") ||
		strings.Contains(ct, "text/xml")
}
