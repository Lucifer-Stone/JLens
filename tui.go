package main

import (
	"encoding/json"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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
