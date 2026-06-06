package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Validator validates manifest entities against JSON schemas
type Validator struct {
	schemasDir string
	schemas    map[string]*jsonschema.Schema // kind -> compiled schema
}

// NewValidator creates a new Validator
func NewValidator(schemasDir string) (*Validator, error) {
	v := &Validator{
		schemasDir: schemasDir,
		schemas:    make(map[string]*jsonschema.Schema),
	}

	// Check if schemas directory exists
	if _, err := os.Stat(schemasDir); os.IsNotExist(err) {
		return v, nil // Return empty validator, schemas will be skipped
	}

	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read schemas directory: %s: %w", schemasDir, err)
	}

	c := jsonschema.NewCompiler()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".schema.json") {
			continue
		}

		kind := strings.TrimSuffix(entry.Name(), ".schema.json")
		schemaPath := filepath.Join(schemasDir, entry.Name())

		schemaData, err := os.ReadFile(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read schema file: %s: %w", schemaPath, err)
		}

		var schemaDoc any
		if err := json.Unmarshal(schemaData, &schemaDoc); err != nil {
			return nil, fmt.Errorf("failed to parse schema JSON: %s: %w", schemaPath, err)
		}

		schemaURL := "file:///" + filepath.ToSlash(schemaPath)
		if err := c.AddResource(schemaURL, schemaDoc); err != nil {
			return nil, fmt.Errorf("failed to add schema resource: %s: %w", schemaPath, err)
		}

		schema, err := c.Compile(schemaURL)
		if err != nil {
			return nil, fmt.Errorf("failed to compile schema: %s: %w", schemaPath, err)
		}

		v.schemas[kind] = schema
	}

	return v, nil
}

// ValidateSchema validates an entity against its corresponding JSON schema
func (v *Validator) ValidateSchema(entity *Entity) []ValidationError {
	schema, ok := v.schemas[entity.Kind]
	if !ok {
		// No schema for this kind - skip (WARNING level, not ERROR)
		return nil
	}

	err := schema.Validate(entity.Raw)
	if err == nil {
		return nil
	}

	var errors []ValidationError

	// Extract validation errors
	if validErr, ok := err.(*jsonschema.ValidationError); ok {
		for _, cause := range flattenValidationErrors(validErr) {
			errors = append(errors, ValidationError{
				File:    entity.FilePath,
				Message: fmt.Sprintf("schema validation: %s", cause),
			})
		}
	} else {
		errors = append(errors, ValidationError{
			File:    entity.FilePath,
			Message: fmt.Sprintf("schema validation: %v", err),
		})
	}

	return errors
}

// flattenValidationErrors recursively flattens jsonschema.ValidationError
func flattenValidationErrors(err *jsonschema.ValidationError) []string {
	var messages []string

	if err.ErrorKind != nil {
		location := ""
		if len(err.InstanceLocation) > 0 {
			location = "/" + strings.Join(err.InstanceLocation, "/") + ": "
		}
		messages = append(messages, location+fmt.Sprintf("%v", err.ErrorKind))
	}

	for _, cause := range err.Causes {
		messages = append(messages, flattenValidationErrors(cause)...)
	}

	return messages
}

// ValidateIDUniqueness validates that all entity IDs are globally unique
func ValidateIDUniqueness(entities []*Entity, memDocs []*MemoryDoc) []ValidationError {
	idMap := make(map[string][]string) // id -> list of file paths

	for _, e := range entities {
		idMap[e.ID] = append(idMap[e.ID], e.FilePath)
	}
	for _, d := range memDocs {
		idMap[d.ID] = append(idMap[d.ID], d.FilePath)
	}

	var errors []ValidationError
	for id, files := range idMap {
		if len(files) > 1 {
			errors = append(errors, ValidationError{
				File:    files[0],
				Message: fmt.Sprintf("duplicate id '%s' found in: %s", id, strings.Join(files, ", ")),
			})
		}
	}

	return errors
}

// ValidateReferences validates reference integrity
func ValidateReferences(entities []*Entity, memDocs []*MemoryDoc, rootDir string) []ValidationError {
	var errors []ValidationError

	// Build all entity/doc ID set
	allIDs := make(map[string]bool)
	for _, e := range entities {
		allIDs[e.ID] = true
	}
	for _, d := range memDocs {
		allIDs[d.ID] = true
	}

	// Build capability ID set
	capabilityIDs := make(map[string]bool)
	for _, e := range entities {
		if e.Kind == "capability" {
			capabilityIDs[e.ID] = true
		}
	}

	// Build archDoc ID set
	memDocIDs := make(map[string]bool)
	for _, d := range memDocs {
		memDocIDs[d.ID] = true
	}

	for _, e := range entities {

		// Check uses_capabilities references
		if usesCaps, ok := e.Raw["uses_capabilities"]; ok {
			if capList, ok := usesCaps.([]any); ok {
				for _, cap := range capList {
					capID := fmt.Sprintf("%v", cap)
					if !capabilityIDs[capID] {
						errors = append(errors, ValidationError{
							File:    e.FilePath,
							Message: fmt.Sprintf("uses_capabilities references unknown capability '%s'", capID),
						})
					}
				}
			}
		}

		// Check bundle includes references
		if e.Kind == "bundle" {
			if includesVal, ok := e.Raw["includes"]; ok {
				if includesList, ok := includesVal.([]any); ok {
					for _, inc := range includesList {
						incID := fmt.Sprintf("%v", inc)
						if !allIDs[incID] {
							errors = append(errors, ValidationError{
								File:    e.FilePath,
								Message: fmt.Sprintf("bundle includes references unknown entity '%s'", incID),
							})
						}
					}
				}
			}
		}
	}

	// Check depends_on in memDocs
	for _, d := range memDocs {
		for _, depID := range d.DependsOn {
			if !memDocIDs[depID] {
				errors = append(errors, ValidationError{
					File:    d.FilePath,
					Message: fmt.Sprintf("depends_on references unknown memory document '%s'", depID),
				})
			}
		}
	}

	return errors
}
