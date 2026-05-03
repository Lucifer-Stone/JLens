package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/fatih/color"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-isatty"
	"github.com/pelletier/go-toml/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

var (
	// CLI Flags
	minifyFlag   bool
	sortKeysFlag bool
	indentFlag   int
	colorFlag    string // "auto", "always", "never"
	validateFlag bool
	queryFlag    string
	statsFlag    bool
	diffFlag     string
	outputFlag   string

	// New Feature Flags
	toFlag          string
	maskFlag        string
	schemaFlag      string
	clipFlag        bool
	interactiveFlag bool

	// Syntax Highlighting Colors
	keyColor    = color.New(color.FgCyan, color.Bold)
	stringColor = color.New(color.FgGreen)
	numColor    = color.New(color.FgYellow)
	boolColor   = color.New(color.FgHiMagenta)
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "jlens [file]",
		Short: "JLens is a fast, powerful JSON formatter and query tool.",
		Long:  `JLens parses, formats, queries, and diffs JSON data. It supports stdin piping, file reading, and syntax highlighting.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runJLens,
	}

	// Define Flags
	rootCmd.Flags().BoolVarP(&minifyFlag, "minify", "m", false, "Minify JSON output")
	rootCmd.Flags().BoolVarP(&sortKeysFlag, "sort-keys", "s", false, "Sort JSON keys alphabetically")
	rootCmd.Flags().IntVarP(&indentFlag, "indent", "i", 2, "Number of spaces for indentation")
	rootCmd.Flags().StringVarP(&colorFlag, "color", "c", "auto", "Colorize output: auto, always, never")
	rootCmd.Flags().BoolVarP(&validateFlag, "validate", "v", false, "Only validate JSON, outputting errors if any")
	rootCmd.Flags().StringVarP(&queryFlag, "query", "q", "", "jq-like query to extract data (e.g., 'users.0.name')")
	rootCmd.Flags().BoolVar(&statsFlag, "stats", false, "Show JSON structure statistics")
	rootCmd.Flags().StringVar(&diffFlag, "diff", "", "Compare input JSON with another file")
	rootCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Write output to a file instead of stdout")

	// New Feature Flags
	rootCmd.Flags().StringVar(&toFlag, "to", "", "Convert output to 'yaml' or 'toml'")
	rootCmd.Flags().StringVar(&maskFlag, "mask", "", "Comma-separated keys to mask with ***")
	rootCmd.Flags().StringVar(&schemaFlag, "schema", "", "Path to JSON schema file to validate against")
	rootCmd.Flags().BoolVar(&clipFlag, "clip", false, "Copy output to clipboard")
	rootCmd.Flags().BoolVar(&interactiveFlag, "interactive", false, "Open interactive TUI mode")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runJLens(cmd *cobra.Command, args []string) error {
	// 1. Read Input
	inputData, err := readInput(args)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	if len(inputData) == 0 {
		return fmt.Errorf("no input provided")
	}

	// 1.5 Schema Validation
	if schemaFlag != "" {
		if err := validateSchema(inputData, schemaFlag); err != nil {
			fmt.Println(color.RedString("❌ Schema validation failed:"))
			fmt.Println(err)
			return err
		}
		fmt.Println(color.GreenString("✔ JSON is valid according to schema."))
	}

	// 2. Validate JSON & Get Errors
	var jsonObj interface{}
	if err := json.Unmarshal(inputData, &jsonObj); err != nil {
		return handleJSONError(inputData, err)
	}

	if validateFlag {
		fmt.Println(color.GreenString("✔ JSON is valid."))
		return nil
	}

	// 2.5 TUI Mode
	if interactiveFlag {
		return runInteractiveTUI(inputData)
	}

	// 2.6 Masking
	if maskFlag != "" {
		jsonObj = applyMask(jsonObj, maskFlag)
	}

	// 3. Stats Mode
	if statsFlag {
		printStats(jsonObj)
		return nil
	}

	// 4. Diff Mode
	if diffFlag != "" {
		return runDiff(inputData, diffFlag)
	}

	// 5. Query Mode
	if queryFlag != "" {
		result := gjson.GetBytes(inputData, queryFlag)
		if !result.Exists() {
			fmt.Println("Query returned no results.")
			return nil
		}
		// Convert result back to byte array for standard processing
		inputData = []byte(result.Raw)
		json.Unmarshal(inputData, &jsonObj) // Re-parse for formatting
	}

	// 5.5 Format Conversion
	if toFlag != "" {
		converted, err := convertFormat(jsonObj, toFlag)
		if err != nil {
			return err
		}
		outStr := string(converted)

		if clipFlag {
			if err := copyToClipboard(outStr); err != nil {
				fmt.Println(color.RedString("Failed to copy to clipboard: " + err.Error()))
			} else {
				fmt.Println(color.GreenString("✔ Copied to clipboard!"))
			}
		}

		if outputFlag != "" {
			os.WriteFile(outputFlag, converted, 0644)
			fmt.Printf("Output successfully written to %s\n", outputFlag)
			return nil
		}
		fmt.Println(outStr)
		return nil
	}

	// 6. Formatting
	formattedData, err := formatJSON(jsonObj)
	if err != nil {
		return fmt.Errorf("formatting failed: %w", err)
	}

	// 7. Handle Output
	outStr := string(formattedData)
	useColor := shouldColorize()

	if outputFlag != "" {
		err := os.WriteFile(outputFlag, formattedData, 0644)
		if err != nil {
			return fmt.Errorf("failed to write to output file: %w", err)
		}
		fmt.Printf("Output successfully written to %s\n", outputFlag)
		return nil
	}

	if useColor && !minifyFlag {
		outStr = applySyntaxHighlighting(outStr)
	}

	if clipFlag {
		if err := copyToClipboard(outStr); err != nil {
			fmt.Println(color.RedString("Failed to copy to clipboard: " + err.Error()))
		} else {
			fmt.Println(color.GreenString("✔ Copied to clipboard!"))
		}
	}

	fmt.Println(outStr)
	return nil
}

// --- Core Functions ---

func readInput(args []string) ([]byte, error) {
	// Read from file if argument is provided
	if len(args) > 0 {
		return os.ReadFile(args[0])
	}

	stat, _ := os.Stdin.Stat()
	// Check if data is piped via stdin
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return io.ReadAll(os.Stdin)
	}

	return nil, nil
}

func handleJSONError(data []byte, err error) error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError

	color.Set(color.FgRed, color.Bold)
	defer color.Unset()

	if string(data)[0] != '{' && string(data)[0] != '[' {
		return fmt.Errorf("invalid JSON: payload does not start with an object or array")
	}

	// Provide line and column context
	if fmt.Sprintf("%T", err) == "*json.SyntaxError" {
		syntaxErr = err.(*json.SyntaxError)
		line, col := getLineAndCol(data, syntaxErr.Offset)
		return fmt.Errorf("JSON Syntax Error at line %d, col %d: %s", line, col, syntaxErr.Error())
	}
	if fmt.Sprintf("%T", err) == "*json.UnmarshalTypeError" {
		typeErr = err.(*json.UnmarshalTypeError)
		line, col := getLineAndCol(data, typeErr.Offset)
		return fmt.Errorf("JSON Type Error at line %d, col %d: expected %s, got %s", line, col, typeErr.Type, typeErr.Value)
	}

	return fmt.Errorf("invalid JSON: %w", err)
}

func getLineAndCol(data []byte, offset int64) (int, int) {
	line := 1
	col := 1
	for i := 0; i < int(offset) && i < len(data); i++ {
		if data[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

func formatJSON(obj interface{}) ([]byte, error) {
	if sortKeysFlag {
		obj = sortMapKeys(obj)
	}

	if minifyFlag {
		return json.Marshal(obj)
	}

	indentStr := strings.Repeat(" ", indentFlag)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // Prevent escaping <, >, &
	enc.SetIndent("", indentStr)

	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

// Recursively sorts map keys
func sortMapKeys(v interface{}) interface{} {
	switch val := v.(type) {
	case []interface{}:
		for i, item := range val {
			val[i] = sortMapKeys(item)
		}
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Use an ordered representation if possible, but standard map doesn't preserve order.
		// Go's json.Marshal automatically sorts map keys anyway!
		// However, returning it triggers Go's default behavior securely.
		for _, k := range keys {
			val[k] = sortMapKeys(val[k])
		}
	}
	return v
}

func shouldColorize() bool {
	if colorFlag == "always" {
		return true
	}
	if colorFlag == "never" {
		return false
	}
	// "auto": check if stdout is a terminal (don't colorize if piped to another command)
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// Uses regex on PRETTY-PRINTED json to apply colors safely
func applySyntaxHighlighting(jsonStr string) string {
	// Key regex: "key":
	keyRe := regexp.MustCompile(`(?m)^(\s*)(".*?")(\s*:)`)
	jsonStr = keyRe.ReplaceAllStringFunc(jsonStr, func(m string) string {
		parts := keyRe.FindStringSubmatch(m)
		return parts[1] + keyColor.Sprint(parts[2]) + parts[3]
	})

	// String value regex: : "value"
	strRe := regexp.MustCompile(`(:\s*)(".*?")(,?)$`)
	jsonStr = strRe.ReplaceAllStringFunc(jsonStr, func(m string) string {
		parts := strRe.FindStringSubmatch(m)
		return parts[1] + stringColor.Sprint(parts[2]) + parts[3]
	})

	// Number value regex
	numRe := regexp.MustCompile(`(:\s*)([-+]?\d*\.?\d+([eE][-+]?\d+)?)(,?)$`)
	jsonStr = numRe.ReplaceAllStringFunc(jsonStr, func(m string) string {
		parts := numRe.FindStringSubmatch(m)
		return parts[1] + numColor.Sprint(parts[2]) + parts[4]
	})

	// Boolean / Null value regex
	boolRe := regexp.MustCompile(`(:\s*)(true|false|null)(,?)$`)
	jsonStr = boolRe.ReplaceAllStringFunc(jsonStr, func(m string) string {
		parts := boolRe.FindStringSubmatch(m)
		return parts[1] + boolColor.Sprint(parts[2]) + parts[3]
	})

	return jsonStr
}

// --- Advanced Features ---

func printStats(obj interface{}) {
	var totalKeys, totalArrays, totalObjects, maxDepth int

	var walk func(v interface{}, currentDepth int)
	walk = func(v interface{}, currentDepth int) {
		if currentDepth > maxDepth {
			maxDepth = currentDepth
		}
		switch val := v.(type) {
		case map[string]interface{}:
			totalObjects++
			totalKeys += len(val)
			for _, child := range val {
				walk(child, currentDepth+1)
			}
		case []interface{}:
			totalArrays++
			for _, child := range val {
				walk(child, currentDepth+1)
			}
		}
	}

	walk(obj, 0)

	fmt.Println(color.CyanString("📊 JSON Statistics"))
	fmt.Println(strings.Repeat("-", 20))
	fmt.Printf("Total Objects: %d\n", totalObjects)
	fmt.Printf("Total Arrays:  %d\n", totalArrays)
	fmt.Printf("Total Keys:    %d\n", totalKeys)
	fmt.Printf("Max Depth:     %d\n", maxDepth)
}

// --- Diff Engine ---

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

// --- Format Conversion Engine ---

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
		b, err := toml.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to TOML: %w", err)
		}
		return b, nil
	default:
		return nil, fmt.Errorf("unsupported format '%s', available formats: yaml, toml", format)
	}
}

// --- Masking Engine ---

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

// --- Schema Engine ---

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

// --- Clipboard Engine ---

func copyToClipboard(text string) error {
	err := clipboard.WriteAll(text)
	if err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	return nil
}

// --- TUI Engine ---

func runInteractiveTUI(jsonData []byte) error {
	var jsonObj interface{}
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		return fmt.Errorf("invalid JSON for TUI: %w", err)
	}

	app := tview.NewApplication()

	tree := tview.NewTreeView().
		SetRoot(tview.NewTreeNode("root").SetColor(tcell.ColorRed)).
		SetCurrentNode(tview.NewTreeNode("root"))

	addNode(tree.GetRoot(), jsonObj)

	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference == nil {
			return // Selecting the root node does nothing.
		}
		children := node.GetChildren()
		if len(children) == 0 {
			// Load and show children on first selection.
			addNode(node, reference)
		} else {
			// Collapse if visible, expand if collapsed.
			node.SetExpanded(!node.IsExpanded())
		}
	})

	layout := tview.NewFlex().
		AddItem(tree, 0, 1, true)

	app.SetRoot(layout, true)

	if err := app.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

func addNode(target *tview.TreeNode, value interface{}) {
	switch v := value.(type) {
	case map[string]interface{}:
		for k, val := range v {
			node := tview.NewTreeNode(k).
				SetReference(val).
				SetSelectable(true).
				SetColor(tcell.ColorAqua)

			if isLeaf(val) {
				node.SetText(fmt.Sprintf("%s: %v", k, formatLeaf(val)))
				node.SetColor(tcell.ColorGreen)
			}
			target.AddChild(node)
		}
	case []interface{}:
		for i, val := range v {
			node := tview.NewTreeNode(fmt.Sprintf("[%d]", i)).
				SetReference(val).
				SetSelectable(true).
				SetColor(tcell.ColorYellow)

			if isLeaf(val) {
				node.SetText(fmt.Sprintf("[%d]: %v", i, formatLeaf(val)))
				node.SetColor(tcell.ColorGreen)
			}
			target.AddChild(node)
		}
	}
}

func isLeaf(value interface{}) bool {
	switch value.(type) {
	case map[string]interface{}, []interface{}:
		return false
	default:
		return true
	}
}

func formatLeaf(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}
