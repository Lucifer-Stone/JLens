package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/fatih/color"
)

type DiffType int

const (
	DiffAdd DiffType = iota
	DiffRemove
	DiffModify
)

type DiffResult struct {
	Type     DiffType
	Path     string
	OldValue interface{}
	NewValue interface{}
}

func runDiff(data1 []byte, file2 string) error {
	data2, err := os.ReadFile(file2)
	if err != nil {
		return fmt.Errorf("failed to read diff file: %w", err)
	}

	var obj1, obj2 interface{}
	if err := json.Unmarshal(data1, &obj1); err != nil {
		return fmt.Errorf("failed to parse input JSON: %w", err)
	}
	if err := json.Unmarshal(data2, &obj2); err != nil {
		return fmt.Errorf("failed to parse target diff JSON: %w", err)
	}

	diffs := compareJSON("root", obj1, obj2)

	if len(diffs) == 0 {
		fmt.Println(color.GreenString("✔ JSON payloads are structurally identical."))
		return nil
	}

	fmt.Println(color.YellowString("⚠ Differences detected:\n"))

	for _, d := range diffs {
		pathStr := color.CyanString(d.Path)
		switch d.Type {
		case DiffModify:
			fmt.Printf("[~] %s: %s\n", color.YellowString("MODIFIED"), pathStr)
			oldStr, _ := formatJSONFast(d.OldValue)
			newStr, _ := formatJSONFast(d.NewValue)
			fmt.Printf("    %s\n", color.RedString("- "+oldStr))
			fmt.Printf("    %s\n", color.GreenString("+ "+newStr))
		case DiffAdd:
			fmt.Printf("[+] %s: %s\n", color.GreenString("ADDED   "), pathStr)
			newStr, _ := formatJSONFast(d.NewValue)
			fmt.Printf("    %s\n", color.GreenString("+ "+newStr))
		case DiffRemove:
			fmt.Printf("[-] %s: %s\n", color.RedString("DELETED "), pathStr)
			oldStr, _ := formatJSONFast(d.OldValue)
			fmt.Printf("    %s\n", color.RedString("- "+oldStr))
		}
		fmt.Println()
	}

	return nil
}

func compareJSON(path string, a, b interface{}) []DiffResult {
	var results []DiffResult

	if a == nil && b == nil {
		return results
	}
	if a == nil {
		return []DiffResult{{Type: DiffAdd, Path: path, NewValue: b}}
	}
	if b == nil {
		return []DiffResult{{Type: DiffRemove, Path: path, OldValue: a}}
	}

	valA := reflect.ValueOf(a)
	valB := reflect.ValueOf(b)

	if valA.Type() != valB.Type() {
		return []DiffResult{{Type: DiffModify, Path: path, OldValue: a, NewValue: b}}
	}

	switch valA.Kind() {
	case reflect.Map:
		mapA := a.(map[string]interface{})
		mapB := b.(map[string]interface{})

		keysA := getKeys(mapA)
		keysB := getKeys(mapB)

		allKeys := make(map[string]bool)
		for _, k := range keysA {
			allKeys[k] = true
		}
		for _, k := range keysB {
			allKeys[k] = true
		}

		sortedKeys := make([]string, 0, len(allKeys))
		for k := range allKeys {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)

		for _, key := range sortedKeys {
			subPath := key
			if path != "" && path != "root" {
				subPath = path + "." + key
			} else if path == "root" {
				subPath = key
			}

			valInA, inA := mapA[key]
			valInB, inB := mapB[key]

			if inA && !inB {
				results = append(results, DiffResult{Type: DiffRemove, Path: subPath, OldValue: valInA})
			} else if !inA && inB {
				results = append(results, DiffResult{Type: DiffAdd, Path: subPath, NewValue: valInB})
			} else {
				results = append(results, compareJSON(subPath, valInA, valInB)...)
			}
		}

	case reflect.Slice:
		sliceA := a.([]interface{})
		sliceB := b.([]interface{})

		maxLen := len(sliceA)
		if len(sliceB) > maxLen {
			maxLen = len(sliceB)
		}

		for i := 0; i < maxLen; i++ {
			subPath := fmt.Sprintf("%s[%d]", path, i)
			if path == "root" {
				subPath = fmt.Sprintf("[%d]", i)
			}

			if i < len(sliceA) && i >= len(sliceB) {
				results = append(results, DiffResult{Type: DiffRemove, Path: subPath, OldValue: sliceA[i]})
			} else if i >= len(sliceA) && i < len(sliceB) {
				results = append(results, DiffResult{Type: DiffAdd, Path: subPath, NewValue: sliceB[i]})
			} else {
				results = append(results, compareJSON(subPath, sliceA[i], sliceB[i])...)
			}
		}

	default:
		if !reflect.DeepEqual(a, b) {
			results = append(results, DiffResult{Type: DiffModify, Path: path, OldValue: a, NewValue: b})
		}
	}

	return results
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func formatJSONFast(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
