# 🔍 JLens — Fast JSON Formatter for the Terminal

JLens is a fast, lightweight, CLI-first JSON formatter built for developers who live in the terminal.

If you use `curl` and deal with messy JSON responses, JLens turns chaos into clarity.

---

## ⚡ Features

* 🧼 Pretty-print JSON
* 📦 Minify JSON
* 🔤 Sort keys
* 🎨 Syntax highlighting
* ✅ JSON validation with error location
* 🔍 Query JSON (jq-like)
* 📊 JSON statistics
* 🔁 Diff two JSON files
* 📥 Works with stdin, files, or raw input

---

## 🚀 Installation

```bash
go install github.com/Lucifer-Stone/jlens@latest
```

Make sure `$GOPATH/bin` is in your PATH.

---

## 🧪 Usage

### Format JSON from curl

```bash
curl https://api.example.com/users | jlens
```

---

### Format a file

```bash
jlens data.json
```

---

### Minify JSON

```bash
jlens --minify data.json
```

---

### Sort Keys

```bash
jlens --sort-keys data.json
```

---

### Query JSON

```bash
jlens --query "users.0.name" data.json
```

---

### Validate JSON

```bash
jlens --validate data.json
```

---

### JSON Stats

```bash
jlens --stats data.json
```

---

### Diff JSON

```bash
jlens --diff file1.json file2.json
```

---

## 🎯 Example

```bash
curl https://api.example.com/users | jlens --query "users.0.name"
```

---

## 🧠 Why JLens?

Unlike tools like jq, JLens focuses on:

* Simplicity
* Readability
* Developer experience

---

## 🛠️ Build from Source

```bash
git clone https://github.com/Lucifer-Stone/jlens.git
cd jlens
go build -o jlens
```

---

## 🗺️ Roadmap

* [ ] Interactive TUI (`--interactive`)
* [ ] Advanced JSON diff viewer
* [ ] Streaming parser for huge JSON
* [ ] Plugin system
* [ ] AI-powered JSON insights

---

## 🤝 Contributing

Pull requests are welcome! For major changes, please open an issue first.

---

## 📄 License

MIT License
