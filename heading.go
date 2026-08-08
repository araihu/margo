package margo

import "github.com/yuin/goldmark/ast"

func collectHeadingIDs(root ast.Node) []string {
	if root == nil {
		return nil
	}
	ids := make([]string, 0)
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if heading, ok := node.(*ast.Heading); ok {
			if value, exists := heading.AttributeString("id"); exists {
				if id, ok := value.([]byte); ok {
					ids = append(ids, string(id))
				} else if id, ok := value.(string); ok {
					ids = append(ids, id)
				}
			}
		}
		return ast.WalkContinue, nil
	})
	return ids
}

func headingIDsForTest(document *Document) []string {
	if document == nil {
		return nil
	}
	parsed, ok := document.parsed.(normalizedMarkdown)
	if !ok {
		return nil
	}
	return append([]string(nil), parsed.headingIDs...)
}
