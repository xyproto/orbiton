package main

import (
	"os"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/xyproto/vt"
)

// exportMarkdownHTML will render HTML from Markdown using the gomarkdown package
func (e *Editor) exportMarkdownHTML(c *vt.Canvas, status *StatusBar, htmlFilename string) error {
	status.ClearAll(c, true)
	status.SetMessage("Rendering to HTML using gomarkdown...")
	status.ShowNoTimeout(c, e)

	// Get the current content of the editor
	mdContent := e.String()

	// Create a Markdown parser with the desired extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	mdParser := parser.NewWithExtensions(extensions)

	// Convert the Markdown content to HTML
	htmlBytes := markdown.ToHTML([]byte(mdContent), mdParser, nil)

	// Write the HTML content to the output file
	err := os.WriteFile(htmlFilename, htmlBytes, 0o644)
	if err != nil {
		status.ClearAll(c, false)
		status.SetError(err)
		status.Show(c, e)
		return err
	}

	status.ClearAll(c, true)
	status.SetMessage("Saved " + htmlFilename)
	status.ShowNoTimeout(c, e)
	return nil
}
