package main

import (
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// validateSchema validates JSON data against a JSON schema file
func validateSchema(jsonData []byte, schemaFile string) error {
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaFile)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	if result.Valid() {
		return nil
	}

	var errMsgs []string
	for _, desc := range result.Errors() {
		errMsgs = append(errMsgs, fmt.Sprintf("- %s", desc))
	}
	return fmt.Errorf("schema validation failed:\n%s", stringsJoin(errMsgs, "\n"))
}

func stringsJoin(arr []string, sep string) string {
	res := ""
	for i, s := range arr {
		if i > 0 {
			res += sep
		}
		res += s
	}
	return res
}
