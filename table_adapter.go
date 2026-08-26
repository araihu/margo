package margo

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/a-h/templ"
	goshtosotable "github.com/araihu/goshtoso/components/table"
	goldast "github.com/yuin/goldmark/ast"
	tableast "github.com/yuin/goldmark/extension/ast"
)

// TableSortMode is the bounded sorting vocabulary exposed by the root
// renderer. Server-side table behavior is deliberately not part of C5.
type TableSortMode string

const (
	TableSortClient TableSortMode = "client"
)

// WithTableSort selects the table sorting projection for one render.
func WithTableSort(mode TableSortMode) RenderOption {
	return func(options *renderOptions) error {
		if mode != TableSortClient {
			return fmt.Errorf("table.sort_mode_invalid: only client sorting is supported")
		}
		options.values["tableSortMode"] = mode
		return nil
	}
}

func tableSortMode(options renderOptions) TableSortMode {
	if value, ok := options.values["tableSortMode"].(TableSortMode); ok {
		return value
	}
	return TableSortClient
}

func renderMarkdownTable(renderer markdownRenderer, node *tableast.Table, mode TableSortMode, id string) error {
	if mode != TableSortClient {
		return fmt.Errorf("table.sort_mode_invalid: only client sorting is supported")
	}
	ctx, out, source := renderer.ctx, renderer.out, renderer.source
	header, ok := node.FirstChild().(*tableast.TableHeader)
	if !ok {
		return fmt.Errorf("table.header_missing: table header is required")
	}
	columns := make([]goshtosotable.Column, 0, header.ChildCount())
	for index, child := 0, header.FirstChild(); child != nil; index, child = index+1, child.NextSibling() {
		cell, ok := child.(*tableast.TableCell)
		if !ok {
			return fmt.Errorf("table.cell_invalid: table header contains an invalid cell")
		}
		columns = append(columns, goshtosotable.Column{
			Key:   fmt.Sprintf("column-%d", index),
			Label: plainInlineText(cell, source),
		})
	}
	if len(columns) == 0 {
		return fmt.Errorf("table.columns_empty: table requires at least one column")
	}
	rows := make([]goshtosotable.Row, 0)
	rowNumber := 0
	for child := header.NextSibling(); child != nil; child = child.NextSibling() {
		row, ok := child.(*tableast.TableRow)
		if !ok {
			continue
		}
		cells := make(map[string]goshtosotable.Cell, len(columns))
		for index, cellNode := 0, row.FirstChild(); index < len(columns); index, cellNode = index+1, cellNode.NextSibling() {
			if cellNode == nil {
				return fmt.Errorf("table.row_width: row %d has fewer cells than the header", rowNumber+1)
			}
			cell, ok := cellNode.(*tableast.TableCell)
			if !ok {
				return fmt.Errorf("table.cell_invalid: row contains an invalid cell")
			}
			cells[columns[index].Key] = markdownTableCell(renderer, cell)
		}
		if cellNode := row.FirstChild(); cellNode != nil {
			count := 0
			for current := cellNode; current != nil; current = current.NextSibling() {
				count++
			}
			if count > len(columns) {
				return fmt.Errorf("table.row_width: row %d has more cells than the header", rowNumber+1)
			}
		}
		rows = append(rows, goshtosotable.Row{ID: fmt.Sprintf("row-%d", rowNumber), Cells: cells})
		rowNumber++
	}
	cfg := goshtosotable.Config{
		ID:        id,
		Caption:   "Markdown table",
		Columns:   columns,
		Rows:      rows,
		RootClass: "margo-table",
	}
	if _, err := io.WriteString(out, `<div data-table-client-sort="true" data-margo-table-sort="natural">`); err != nil {
		return err
	}
	if err := goshtosotable.Table(cfg).Render(ctx, out); err != nil {
		return err
	}
	_, err := io.WriteString(out, `</div>`)
	return err
}

func markdownTableCell(renderer markdownRenderer, cell *tableast.TableCell) goshtosotable.Cell {
	result := goshtosotable.Cell{Text: plainInlineText(cell, renderer.source)}
	if !inlineContainsLink(cell) {
		return result
	}
	result.Component = templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		cellRenderer := renderer
		cellRenderer.ctx = ctx
		cellRenderer.out = out
		return cellRenderer.renderInlineChildren(cell)
	})
	return result
}

func inlineContainsLink(node goldast.Node) bool {
	found := false
	_ = goldast.Walk(node, func(current goldast.Node, entering bool) (goldast.WalkStatus, error) {
		if !entering {
			return goldast.WalkContinue, nil
		}
		switch current.(type) {
		case *goldast.Link, *goldast.AutoLink:
			found = true
			return goldast.WalkStop, nil
		default:
			return goldast.WalkContinue, nil
		}
	})
	return found
}

func plainInlineText(node goldast.Node, source []byte) string {
	var builder strings.Builder
	_ = goldast.Walk(node, func(current goldast.Node, entering bool) (goldast.WalkStatus, error) {
		if !entering {
			return goldast.WalkContinue, nil
		}
		switch value := current.(type) {
		case *goldast.Text:
			builder.Write(value.Value(source))
			if value.HardLineBreak() || value.SoftLineBreak() {
				builder.WriteByte(' ')
			}
		case *goldast.String:
			builder.Write(value.Value)
		case *goldast.AutoLink:
			builder.Write(value.Label(source))
		case *goldast.CodeSpan:
			for child := value.FirstChild(); child != nil; child = child.NextSibling() {
				text := child.(*goldast.Text).Segment.Value(source)
				if bytes.HasSuffix(text, []byte("\n")) {
					builder.Write(text[:len(text)-1])
					builder.WriteByte(' ')
				} else {
					builder.Write(text)
				}
			}
			return goldast.WalkSkipChildren, nil
		case *goldast.RawHTML:
			builder.WriteString(html.UnescapeString(string(value.Segments.Value(source))))
		}
		return goldast.WalkContinue, nil
	})
	return strings.TrimSpace(builder.String())
}

var _ templ.Component = goshtosotable.Instance{}
