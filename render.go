package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xyproto/burnfont"
)

// SavePNG will render the current text to a .png image, using a fixed font
func (e *Editor) SavePNG(filename string) error {

	// Find the longest line
	maxlen := 0
	for i := 0; i < e.Len(); i++ {
		line := e.Line(i)
		if len(line) > maxlen {
			maxlen = len(line)
		}
	}

	lineHeight := 14
	marginRight := 4 * lineHeight
	width := 8*(maxlen+1) + marginRight
	height := (e.Len() * lineHeight) + 2*lineHeight

	dimension := image.Rectangle{image.Point{}, image.Point{width, height}}

	textImage := image.NewRGBA(dimension)
	finalImage := image.NewRGBA(dimension)

	darkgray := color.NRGBA{0x10, 0x10, 0x10, 0xff}
	white := color.NRGBA{0xff, 0xff, 0xff, 0xff}

	draw.Draw(finalImage, finalImage.Bounds(), &image.Uniform{white}, image.Point{}, draw.Src)

	// For each line of this text document, draw the string to an image
	var contents string
	for i := 0; i < e.Len(); i++ {
		// Expand tabs for each line
		contents = strings.Replace(e.Line(i), "\t", strings.Repeat(" ", e.spacesPerTab), -1)
		// Draw the string to the textImage
		burnfont.DrawString(textImage, lineHeight, (i+1)*lineHeight, contents, darkgray)
	}

	// Now overlay the text image on top of the final image with the background color
	draw.Draw(finalImage, finalImage.Bounds(), textImage, image.Point{}, draw.Over)

	// Write the PNG file
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	return png.Encode(f, finalImage)
}

// SavePDF can save the text as a PDF. It's pretty experimental.
func (e *Editor) SavePDF(filename string) error {
	// For each line of this text document, draw the string to an image
	var sb strings.Builder
	for i := 0; i < e.Len(); i++ {
		// Expand tabs for each line
		sb.WriteString(strings.Replace(e.Line(i), "\t", strings.Repeat(" ", e.spacesPerTab), -1) + "\n")
	}

	contents := sb.String()

	timestamp := time.Now().Format("2006-01-02")

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTopMargin(30)
	topLeftText := "1/1"
	topRightText := timestamp
	pdf.SetHeaderFunc(func() {
		pdf.SetY(5)
		pdf.SetFont("Helvetica", "", 6)
		pdf.CellFormat(80, 0, topLeftText, "", 0, "L", false, 0, "")
		pdf.CellFormat(0, 0, topRightText, "", 0, "R", false, 0, "")
	})
	pdf.AddPage()
	pdf.SetY(20)
	pdf.SetFont("Courier", "B", 12)
	pdf.Write(5, filename+"\n\n")
	pdf.SetFont("Courier", "", 6)
	pdf.Write(5, contents+"\n")

	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return fmt.Errorf("%s already exists!\n", filename)
	}
	//fmt.Printf("Writing %s... ", filename)
	if err := pdf.OutputFileAndClose(filename); err != nil {
		return err
		//fmt.Fprintf(os.Stderr, "%s\n", err)
		//os.Exit(1)
	}
	//fmt.Println("done.")
	return nil
}
