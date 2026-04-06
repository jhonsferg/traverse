package traverse

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AnnotatedEntity wraps a decoded entity and exposes its OData instance annotations.
//
// OData responses may include annotation properties alongside entity data. Annotations
// are any property whose key begins with '@', such as '@odata.etag', '@odata.count',
// or custom vocabulary terms like '@Custom.Score'.
//
// Example usage:
//
//	result, err := DecodeAnnotated[Product](data)
//	if err != nil { ... }
//	fmt.Println(result.Entity.Name)
//	var etag string
//	_ = result.GetAnnotation("@odata.etag", &etag)
type AnnotatedEntity[T any] struct {
	// Entity holds the decoded entity value.
	Entity T
	// Annotations holds all '@'-prefixed annotation properties as raw JSON.
	Annotations map[string]json.RawMessage
}

// DecodeAnnotated decodes an OData JSON object, separating '@'-prefixed annotation
// properties from the regular entity fields.
//
// The raw JSON object is first unmarshalled into a map. Keys beginning with '@' are
// stored as Annotations; the remaining keys are re-encoded and decoded into T.
func DecodeAnnotated[T any](data []byte) (AnnotatedEntity[T], error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return AnnotatedEntity[T]{}, fmt.Errorf("traverse: DecodeAnnotated unmarshal: %w", err)
	}

	annotations := make(map[string]json.RawMessage)
	entity := make(map[string]json.RawMessage, len(raw))

	for k, v := range raw {
		if strings.HasPrefix(k, "@") {
			annotations[k] = v
		} else {
			entity[k] = v
		}
	}

	entityBytes, err := json.Marshal(entity)
	if err != nil {
		return AnnotatedEntity[T]{}, fmt.Errorf("traverse: DecodeAnnotated re-encode: %w", err)
	}

	var t T
	if err := json.Unmarshal(entityBytes, &t); err != nil {
		return AnnotatedEntity[T]{}, fmt.Errorf("traverse: DecodeAnnotated decode entity: %w", err)
	}

	return AnnotatedEntity[T]{
		Entity:      t,
		Annotations: annotations,
	}, nil
}

// GetAnnotation retrieves a typed annotation value by name and unmarshals it into target.
//
// Returns an error if the annotation is not present or cannot be decoded into target.
//
// Example:
//
//	var etag string
//	if err := result.GetAnnotation("@odata.etag", &etag); err != nil { ... }
func (a *AnnotatedEntity[T]) GetAnnotation(name string, target any) error {
	raw, ok := a.Annotations[name]
	if !ok {
		return fmt.Errorf("traverse: annotation %q not found", name)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("traverse: GetAnnotation %q: %w", name, err)
	}
	return nil
}
