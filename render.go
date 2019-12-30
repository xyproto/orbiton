package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	//"github.com/xyproto/burnfont"
)

//// SavePNG will render the current text to a .png image, using a fixed font
//func (e *Editor) SavePNG(filename string) error {
//
//	// Find the longest line
//	maxlen := 0
//	for i := 0; i < e.Len(); i++ {
//		line := e.Line(i)
//		if len(line) > maxlen {
//			maxlen = len(line)
//		}
//	}
//
//	lineHeight := 14
//	marginRight := 4 * lineHeight
//	width := 8*(maxlen+1) + marginRight
//	height := (e.Len() * lineHeight) + 2*lineHeight
//
//	dimension := image.Rectangle{image.Point{}, image.Point{width, height}}
//
//	textImage := image.NewRGBA(dimension)
//	finalImage := image.NewRGBA(dimension)
//
//	darkgray := color.NRGBA{0x10, 0x10, 0x10, 0xff}
//	white := color.NRGBA{0xff, 0xff, 0xff, 0xff}
//
//	draw.Draw(finalImage, finalImage.Bounds(), &image.Uniform{white}, image.Point{}, draw.Src)
//
//	// For each line of this text document, draw the string to an image
//	var contents string
//	for i := 0; i < e.Len(); i++ {
//		// Expand tabs for each line
//		contents = strings.Replace(e.Line(i), "\t", strings.Repeat(" ", e.spacesPerTab), -1)
//		// Draw the string to the textImage
//		burnfont.DrawString(textImage, lineHeight, (i+1)*lineHeight, contents, darkgray)
//	}
//
//	// Now overlay the text image on top of the final image with the background color
//	draw.Draw(finalImage, finalImage.Bounds(), textImage, image.Point{}, draw.Over)
//
//	// Write the PNG file
//	f, err := os.Create(filename)
//	if err != nil {
//		return err
//	}
//	return png.Encode(f, finalImage)
//}

// SavePDF can save the text as a PDF. It's pretty experimental.
func (e *Editor) SavePDF(title, filename string) error {
	// For each line of this text document, draw the string to an image
	var sb strings.Builder
	for i := 0; i < e.Len(); i++ {
		// Expand tabs for each line
		sb.WriteString(strings.Replace(e.Line(i), "\t", strings.Repeat(" ", e.spacesPerTab), -1) + "\n")
	}

	contents := sb.String()

	// Create a timestamp for the current date, using the "2006-01-02" format
	timestamp := time.Now().Format("2006-01-02")

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTopMargin(30)
	pdf.SetHeaderFunc(func() {
		pdf.SetY(5)
		pdf.SetFont("Helvetica", "", 6)
		// Top right corner
		pdf.CellFormat(0, 0, timestamp, "", 0, "R", false, 0, "")
	})
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Helvetica", "", 6)
		// Bottom center
		pdf.CellFormat(0, 10, fmt.Sprintf("%d", pdf.PageNo()), "", 0, "C", false, 0, "")
	})
	pdf.AddPage()
	pdf.SetY(20)
	pdf.SetFont("Courier", "B", 12)
	pdf.Write(5, title+"\n\n")
	pdf.SetFont("Courier", "", 6)
	pdf.Write(5, contents+"\n")

	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return fmt.Errorf("%s already exists", filename)
	}
	if err := pdf.OutputFileAndClose(filename); err != nil {
		return err
	}
	return nil
}
