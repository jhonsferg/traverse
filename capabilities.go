package traverse

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// EntityCapabilities describes what operations are supported for an entity set.
type EntityCapabilities struct {
	Filterable                     bool
	NonFilterableProperties        []string
	Sortable                       bool
	NonSortableProperties          []string
	ExpandableNavigationProperties []string
	Insertable                     bool
	Updatable                      bool
	Deletable                      bool
}

// CapabilityError is returned when a requested operation is not supported.
type CapabilityError struct {
	EntitySet string
	Operation string
	Property  string
	Message   string
}

// Error implements the error interface.
func (e *CapabilityError) Error() string {
	if e.Property != "" {
		return fmt.Sprintf("traverse: capability error on %s: %s operation not supported for property '%s': %s",
			e.EntitySet, e.Operation, e.Property, e.Message)
	}
	return fmt.Sprintf("traverse: capability error on %s: %s operation not supported: %s",
		e.EntitySet, e.Operation, e.Message)
}

// CapabilitiesRegistry holds parsed capabilities for all entity sets.
type CapabilitiesRegistry struct {
	sets map[string]EntityCapabilities
}

// NewCapabilitiesRegistry creates a new empty registry.
func NewCapabilitiesRegistry() *CapabilitiesRegistry {
	return &CapabilitiesRegistry{
		sets: make(map[string]EntityCapabilities),
	}
}

// Get returns the capabilities for the named entity set.
func (r *CapabilitiesRegistry) Get(entitySet string) EntityCapabilities {
	if cap, exists := r.sets[entitySet]; exists {
		return cap
	}
	return EntityCapabilities{
		Filterable: true,
		Sortable:   true,
		Insertable: true,
		Updatable:  true,
		Deletable:  true,
	}
}

// ParseCapabilities parses an OData v4 EDMX metadata document.
func ParseCapabilities(edmxXML []byte) (*CapabilitiesRegistry, error) {
	var root xmlRoot
	err := xml.Unmarshal(edmxXML, &root)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EDMX XML: %w", err)
	}

	registry := NewCapabilitiesRegistry()

	for _, schema := range root.DataServices.Schemas {
		for _, container := range schema.EntityContainers {
			for _, es := range container.EntitySets {
				cap := EntityCapabilities{
					Filterable: true,
					Sortable:   true,
					Insertable: true,
					Updatable:  true,
					Deletable:  true,
				}

				for _, ann := range es.Annotations {
					parseCapabilityTerms(ann, &cap)
				}

				registry.sets[es.Name] = cap
			}
		}

		for _, ann := range schema.Annotations {
			if !strings.HasPrefix(ann.Term, "Capabilities.") {
				continue
			}

			target := ann.Target
			containerPrefix := ""
			if len(schema.EntityContainers) > 0 {
				containerPrefix = schema.EntityContainers[0].Name + "/"
			}
			entitySetName := strings.TrimPrefix(target, containerPrefix)

			if _, exists := registry.sets[entitySetName]; !exists {
				registry.sets[entitySetName] = EntityCapabilities{
					Filterable: true,
					Sortable:   true,
					Insertable: true,
					Updatable:  true,
					Deletable:  true,
				}
			}

			cap := registry.sets[entitySetName]
			parseCapabilityTerms(ann, &cap)
			registry.sets[entitySetName] = cap
		}
	}

	return registry, nil
}

type xmlRoot struct {
	XMLName      xml.Name `xml:"Edmx"`
	DataServices struct {
		Schemas []struct {
			XMLName          xml.Name `xml:"Schema"`
			Namespace        string   `xml:"Namespace,attr"`
			EntityContainers []struct {
				XMLName    xml.Name `xml:"EntityContainer"`
				Name       string   `xml:"Name,attr"`
				EntitySets []struct {
					XMLName     xml.Name        `xml:"EntitySet"`
					Name        string          `xml:"Name,attr"`
					Annotations []xmlAnnotation `xml:"Annotation"`
				} `xml:"EntitySet"`
			} `xml:"EntityContainer"`
			Annotations []xmlAnnotation `xml:"Annotation"`
		} `xml:"Schema"`
	} `xml:"DataServices"`
}

type xmlAnnotation struct {
	XMLName xml.Name `xml:"Annotation"`
	Term    string   `xml:"Term,attr"`
	Target  string   `xml:"Target,attr"`
	Bool    *bool    `xml:"Bool,attr"`
	Records []struct {
		XMLName        xml.Name `xml:"Record"`
		PropertyValues []struct {
			XMLName    xml.Name `xml:"PropertyValue"`
			Property   string   `xml:"Property,attr"`
			Bool       *bool    `xml:"Bool,attr"`
			Collection []struct {
				XMLName xml.Name `xml:"Collection"`
				Records []struct {
					XMLName        xml.Name `xml:"Record"`
					PropertyValues []struct {
						XMLName  xml.Name `xml:"PropertyValue"`
						Property string   `xml:"Property,attr"`
						String   string   `xml:"String,attr"`
					} `xml:"PropertyValue"`
				} `xml:"Record"`
			} `xml:"Collection"`
		} `xml:"PropertyValue"`
	} `xml:"Record"`
}

func parseCapabilityTerms(ann xmlAnnotation, cap *EntityCapabilities) {
	for _, rec := range ann.Records {
		for _, pv := range rec.PropertyValues {
			switch ann.Term {
			case "Capabilities.FilterRestrictions":
				if pv.Property == "Filterable" && pv.Bool != nil {
					cap.Filterable = *pv.Bool
				}
				if pv.Property == "NonFilterableProperties" {
					cap.NonFilterableProperties = extractNamesFromPV(pv.Collection)
				}
			case "Capabilities.SortRestrictions":
				if pv.Property == "Sortable" && pv.Bool != nil {
					cap.Sortable = *pv.Bool
				}
				if pv.Property == "NonSortableProperties" {
					cap.NonSortableProperties = extractNamesFromPV(pv.Collection)
				}
			case "Capabilities.ExpandRestrictions":
				if pv.Property == "ExpandableProperties" {
					cap.ExpandableNavigationProperties = extractNamesFromPV(pv.Collection)
				}
			case "Capabilities.InsertRestrictions":
				if pv.Property == "Insertable" && pv.Bool != nil {
					cap.Insertable = *pv.Bool
				}
			case "Capabilities.UpdateRestrictions":
				if pv.Property == "Updatable" && pv.Bool != nil {
					cap.Updatable = *pv.Bool
				}
			case "Capabilities.DeleteRestrictions":
				if pv.Property == "Deletable" && pv.Bool != nil {
					cap.Deletable = *pv.Bool
				}
			}
		}
	}
}

func extractNamesFromPV(collections interface{}) []string {
	var result []string
	// Safely cast and extract names
	switch v := collections.(type) {
	case []interface{}:
		for _, cItem := range v {
			if cMap, ok := cItem.(map[string]interface{}); ok {
				if records, ok := cMap["Records"].([]interface{}); ok {
					for _, rItem := range records {
						if rMap, ok := rItem.(map[string]interface{}); ok {
							if props, ok := rMap["PropertyValues"].([]interface{}); ok {
								for _, pItem := range props {
									if pMap, ok := pItem.(map[string]interface{}); ok {
										if pMap["Property"] == "Name" {
											if s, ok := pMap["String"].(string); ok && s != "" {
												result = append(result, s)
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return result
}

// WithCapabilitiesValidation enables capability checking on the client.
func WithCapabilitiesValidation(registry *CapabilitiesRegistry) Option {
	return func(cfg *clientConfig) error {
		if registry == nil {
			return nil
		}

		cfg.beforeQuery = append(cfg.beforeQuery, func(qb *QueryBuilder) error {
			if qb.filterExpr != "" && !isFilterAllowed(registry, qb.entitySet) {
				return &CapabilityError{
					EntitySet: qb.entitySet,
					Operation: "filter",
					Message:   "the service does not support filtering on this entity set",
				}
			}

			if qb.filterExpr != "" {
				if props := getRestrictedFilterProperties(registry, qb.entitySet); len(props) > 0 {
					for _, prop := range extractPropertiesFromFilter(qb.filterExpr) {
						for _, restricted := range props {
							if prop == restricted {
								return &CapabilityError{
									EntitySet: qb.entitySet,
									Operation: "filter",
									Property:  prop,
									Message:   "this property is not filterable",
								}
							}
						}
					}
				}
			}

			if qb.orderByExpr != "" && !isSortAllowed(registry, qb.entitySet) {
				return &CapabilityError{
					EntitySet: qb.entitySet,
					Operation: "sort",
					Message:   "the service does not support sorting on this entity set",
				}
			}

			if qb.orderByExpr != "" {
				if props := getRestrictedSortProperties(registry, qb.entitySet); len(props) > 0 {
					for _, prop := range extractPropertiesFromOrderBy(qb.orderByExpr) {
						for _, restricted := range props {
							if prop == restricted {
								return &CapabilityError{
									EntitySet: qb.entitySet,
									Operation: "sort",
									Property:  prop,
									Message:   "this property is not sortable",
								}
							}
						}
					}
				}
			}

			for _, exp := range qb.expandProps {
				if expandProps := registry.Get(qb.entitySet).ExpandableNavigationProperties; len(expandProps) > 0 {
					allowed := false
					for _, ep := range expandProps {
						if ep == exp {
							allowed = true
							break
						}
					}
					if !allowed {
						return &CapabilityError{
							EntitySet: qb.entitySet,
							Operation: "expand",
							Property:  exp,
							Message:   "this navigation property cannot be expanded",
						}
					}
				}
			}

			return nil
		})

		return nil
	}
}

func isFilterAllowed(registry *CapabilitiesRegistry, entitySet string) bool {
	return registry.Get(entitySet).Filterable
}

func isSortAllowed(registry *CapabilitiesRegistry, entitySet string) bool {
	return registry.Get(entitySet).Sortable
}

func getRestrictedFilterProperties(registry *CapabilitiesRegistry, entitySet string) []string {
	return registry.Get(entitySet).NonFilterableProperties
}

func getRestrictedSortProperties(registry *CapabilitiesRegistry, entitySet string) []string {
	return registry.Get(entitySet).NonSortableProperties
}

func extractPropertiesFromFilter(filter string) []string {
	var props []string
	tokens, err := tokenizeFilter(filter)
	if err != nil {
		return props
	}

	for i, token := range tokens {
		if i+1 < len(tokens) {
			op := tokens[i+1]
			if isFilterOperator(op) {
				props = append(props, token)
			}
		}
	}
	return props
}

func extractPropertiesFromOrderBy(orderby string) []string {
	var props []string
	parts := strings.Split(orderby, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) > 0 {
			props = append(props, fields[0])
		}
	}
	return props
}

func isFilterOperator(op string) bool {
	operators := map[string]bool{
		"eq":         true,
		"ne":         true,
		"lt":         true,
		"le":         true,
		"gt":         true,
		"ge":         true,
		"contains":   true,
		"startswith": true,
		"endswith":   true,
	}
	return operators[op]
}
