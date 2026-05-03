package main

import (
	"bytes"
	"fmt"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

func convertFormat(obj interface{}, format string) ([]byte, error) {
	switch format {
	case "yaml", "yml":
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		err := enc.Encode(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to YAML: %w", err)
		}
		return buf.Bytes(), nil
	case "toml":
		// go-toml v2 works better with map[string]interface{} than nested interfaces if not cleanly typed,
		// but since unmarshaled JSON is map[string]interface{}, it usually handles it well.
		b, err := toml.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to TOML: %w", err)
		}
		return b, nil
	default:
		return nil, fmt.Errorf("unsupported format '%s', available formats: yaml, toml", format)
	}
}
