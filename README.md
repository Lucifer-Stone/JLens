# 🔍 JLens — Fast JSON Formatter for the Terminal

JLens is a fast, lightweight, CLI-first JSON formatter built for developers who live in the terminal.

If you use `curl` and deal with messy JSON responses, JLens turns chaos into clarity with advanced features like structural diffing, format conversion, and data masking.

---

## ⚡ Features

* 🧼 **Pretty-print JSON**: Configurable indentation and minification
* 🔤 **Sort keys**: Sort your JSON keys alphabetically
* 🎨 **Syntax highlighting**: Beautiful colors natively in your terminal
* 🔍 **Query JSON**: jq-like queries to extract exact data paths
* 📊 **JSON statistics**: Instantly see tree depth, total keys, and array/object counts
* ✨ **[NEW] Semantic Structural Diff**: Visually compare JSON structures instead of reading messy Git-style line diffs
* ✨ **[NEW] Format Conversion**: Instantly pipe JSON into YAML or TOML (`--to yaml`)
* ✨ **[NEW] Data Masking**: Securely mask sensitive data keys (like tokens/passwords) before logging (`--mask`)
* ✨ **[NEW] Schema Validation**: Validate payloads against JSON schema files (`--schema`)
* ✨ **[NEW] Interactive TUI**: Explore massive JSON trees using a keyboard-navigable terminal UI (`--interactive`)
* ✨ **[NEW] Clipboard Support**: Output formatted data directly to your system clipboard (`--clip`)

---

## 🚀 Installation

Install globally using Go:
```bash
go install github.com/Lucifer-Stone/jlens@latest
```
*Note: Make sure your `$GOPATH/bin` is added to your system's PATH.*

---

## 🧪 Usage Examples

### Format JSON from curl
```bash
curl https://api.example.com/users | jlens
```

### Minify & Sort Keys
```bash
jlens --minify --sort-keys data.json
```

### Query JSON (Extract specific path)
```bash
jlens --query "users.0.name" data.json
```

---

## 🔥 Advanced Developer Workflows

### Semantic JSON Diff
See exactly which keys were added, deleted, or modified.
```bash
jlens --diff file2.json file1.json
```

### Convert JSON to YAML or TOML
```bash
jlens --to yaml data.json
jlens --to toml data.json
```

### Mask Sensitive Data
Perfect for sharing logs securely. Provide a comma-separated list of keys to mask.
```bash
jlens --mask "password,token,secret_key" user.json
```

### Interactive TUI Mode
Explore huge JSON payloads interactively. Expand/collapse nodes with arrow keys.
```bash
jlens --interactive huge_response.json
```

### Validate against JSON Schema
```bash
jlens --schema user_schema.json data.json
```

### Copy to Clipboard
Pipe the final, formatted output directly to your system clipboard.
```bash
jlens --query "user.profile" --to yaml --clip data.json
```

---

## 🛠️ Build from Source

```bash
git clone https://github.com/Lucifer-Stone/jlens.git
cd jlens
go build -o jlens
```

---

## 🤝 Contributing

Pull requests are welcome! For major changes, please open an issue first.

---

## 📄 License

MIT License
