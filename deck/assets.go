package deck

import _ "embed"

var (
	//go:embed assets/deck.css
	deckCSS string

	//go:embed assets/deck.js
	deckJavaScript string
)
