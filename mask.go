package main

import (
	"strings"
)

// applyMask recursively finds and replaces specified keys with "***"
func applyMask(obj interface{}, maskKeysStr string) interface{} {
	if maskKeysStr == "" || obj == nil {
		return obj
	}

	keysToMask := strings.Split(maskKeysStr, ",")
	maskMap := make(map[string]bool)
	for _, k := range keysToMask {
		maskMap[strings.TrimSpace(k)] = true
	}

	return maskRecursive(obj, maskMap)
}

func maskRecursive(v interface{}, maskMap map[string]bool) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for k, child := range val {
			if maskMap[k] {
				newMap[k] = "***"
			} else {
				newMap[k] = maskRecursive(child, maskMap)
			}
		}
		return newMap
	case []interface{}:
		newArr := make([]interface{}, len(val))
		for i, child := range val {
			newArr[i] = maskRecursive(child, maskMap)
		}
		return newArr
	}
	return v
}
