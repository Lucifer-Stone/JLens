package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
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
