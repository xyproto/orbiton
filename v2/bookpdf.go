package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"unicode"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"github.com/jung-kurt/gofpdf"
	"github.com/xyproto/files"
	"github.com/xyproto/vt"
)

// Page layout for exported books, in millimeters: the trim size and margins of a technical book
const (
	pdfPageW        = 178.0
	pdfPageH        = 234.0
	pdfMarginInner  = 22.0
	pdfMarginOuter  = 18.0
	pdfMarginTop    = 26.0
	pdfMarginBottom = 20.0
	pdfHeadY        = 15.0
	pdfBodySize     = 10.0
	pdfLineHeight   = 5.0
	pdfParSpace     = 2.4
	pdfCodeSize     = 8.0
	pdfCodeLine     = 3.9
)

type pageKind int

const (
	pageTitle pageKind = iota
	pageFront
	pageOpener
	pageBody
)

type bookRun struct {
	text  string
	style string // "", "B", "I", "BI" or "M" for code
	size  float64
}

type bookWord struct {
	runs      []bookRun
	width     float64
	breakLine bool // a hard line break follows this word
}

type tocEntry struct {
	level int // 0 for chapters, 1 for sections
	num   string
	title string
}

type bookPDF struct {
	pdf          *gofpdf.Fpdf
	tr           func(string) string
	serif, sans  string
	mono         string
	pageW, pageH float64
	y            float64
	kind, next   pageKind
	title        string
	chapNum      int
	secNum       int
	figNum       int
	chapHead     string // running head on left-hand pages
	secHead      string // running head on right-hand pages
	mainStart    int    // the physical page where chapter 1 (page 1) starts
	quote        int    // depth of block quotes, which get a bar in the margin
	dir          string
	toc          []tocEntry
	tocPages     []int
	tocAt        int
}

// bookFontDirs are where the DejaVu fonts may be installed
var bookFontDirs = []string{"/usr/share/fonts/TTF", "/usr/share/fonts/truetype/dejavu", "/usr/share/fonts/dejavu", "/usr/local/share/fonts"}

var chapterLabel = regexp.MustCompile(`(?i)^chapter\s+(\d+)\s*[:.\-–—]?\s*(.*)$`)

func (b *bookPDF) loadFonts() {
	b.serif, b.sans, b.mono = "Times", "Helvetica", "Courier"
	b.tr = b.pdf.UnicodeTranslatorFromDescriptor("")
	fonts := []struct{ family, style, file string }{
		{"BookSerif", "", "DejaVuSerif.ttf"},
		{"BookSerif", "B", "DejaVuSerif-Bold.ttf"},
		{"BookSerif", "I", "DejaVuSerif-Italic.ttf"},
		{"BookSerif", "BI", "DejaVuSerif-BoldItalic.ttf"},
		{"BookSans", "", "DejaVuSans.ttf"},
		{"BookSans", "B", "DejaVuSans-Bold.ttf"},
		{"BookMono", "", "DejaVuSansMono.ttf"},
		{"BookMono", "B", "DejaVuSansMono-Bold.ttf"},
	}
	for _, dir := range bookFontDirs {
		data := make([][]byte, len(fonts))
		ok := true
		for i, f := range fonts {
			var err error
			if data[i], err = os.ReadFile(filepath.Join(dir, f.file)); err != nil {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		for i, f := range fonts {
			b.pdf.AddUTF8FontFromBytes(f.family, f.style, data[i])
		}
		b.serif, b.sans, b.mono = "BookSerif", "BookSans", "BookMono"
		b.tr = func(s string) string { return s }
		return
	}
}

func (b *bookPDF) font(family, style string, size float64) {
	b.pdf.SetFont(family, style, size)
}

// setRunFont selects the font of a run of body text
func (b *bookPDF) setRunFont(style string, size float64) {
	if style == "M" {
		b.font(b.mono, "", size*0.88)
		return
	}
	b.font(b.serif, style, size)
}

func (b *bookPDF) runWidth(r bookRun) float64 {
	b.setRunFont(r.style, r.size)
	return b.pdf.GetStringWidth(b.tr(r.text))
}

func (b *bookPDF) textWidth(family, style string, size float64, s string) float64 {
	b.font(family, style, size)
	return b.pdf.GetStringWidth(b.tr(s))
}

// margins returns the left and right margins of the current page, mirrored like in a printed book
func (b *bookPDF) margins() (float64, float64) {
	if b.pdf.PageNo()%2 == 0 {
		return pdfMarginOuter, pdfMarginInner
	}
	return pdfMarginInner, pdfMarginOuter
}

func (b *bookPDF) measure() float64 {
	left, right := b.margins()
	return b.pageW - left - right
}

func (b *bookPDF) addPage(kind pageKind) {
	b.next = kind
	b.pdf.AddPage()
	b.y = pdfMarginTop
}

func (b *bookPDF) bottom() float64 {
	return b.pageH - pdfMarginBottom
}

// ensure starts a new page unless h millimeters fit on this one
func (b *bookPDF) ensure(h float64) {
	if b.y+h > b.bottom() {
		b.addPage(pageBody)
	}
}

// space adds vertical space, continuing the bar of a block quote
func (b *bookPDF) space(h float64) {
	if b.y+h > b.bottom() {
		b.y = b.bottom()
		return
	}
	b.quoteBar(b.y, h)
	b.y += h
}

func (b *bookPDF) quoteBar(y, h float64) {
	if b.quote == 0 {
		return
	}
	left, _ := b.margins()
	b.pdf.SetFillColor(150, 150, 150)
	for q := 0; q < b.quote; q++ {
		b.pdf.Rect(left+float64(q)*6, y, 0.7, h, "F")
	}
}

func roman(n int) string {
	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"m", "cm", "d", "cd", "c", "xc", "l", "xl", "x", "ix", "v", "iv", "i"}
	var sb strings.Builder
	for i, v := range vals {
		for n >= v {
			sb.WriteString(syms[i])
			n -= v
		}
	}
	return sb.String()
}

// folio is the page number printed on a physical page
func (b *bookPDF) folio(page int) string {
	if b.mainStart > 0 && page >= b.mainStart {
		return strconv.Itoa(page - b.mainStart + 1)
	}
	return roman(page)
}

func (b *bookPDF) header() {
	b.kind = b.next
	if b.kind != pageBody {
		return
	}
	left, right := b.margins()
	page := b.pdf.PageNo()
	num := b.folio(page)
	head := b.chapHead
	if page%2 == 1 && b.secHead != "" {
		head = b.secHead
	}
	b.pdf.SetTextColor(90, 90, 90)
	b.font(b.sans, "B", 8)
	nw := b.pdf.GetStringWidth(num)
	b.font(b.sans, "", 8)
	hw := b.pdf.GetStringWidth(b.tr(head))
	if page%2 == 0 {
		b.font(b.sans, "B", 8)
		b.pdf.Text(left, pdfHeadY, num)
		b.font(b.sans, "", 8)
		b.pdf.Text(left+nw+4, pdfHeadY, b.tr(head))
	} else {
		b.font(b.sans, "", 8)
		b.pdf.Text(b.pageW-right-nw-4-hw, pdfHeadY, b.tr(head))
		b.font(b.sans, "B", 8)
		b.pdf.Text(b.pageW-right-nw, pdfHeadY, num)
	}
	b.pdf.SetDrawColor(170, 170, 170)
	b.pdf.SetLineWidth(0.2)
	b.pdf.Line(left, pdfHeadY+2.2, b.pageW-right, pdfHeadY+2.2)
	b.pdf.SetTextColor(0, 0, 0)
}

func (b *bookPDF) footer() {
	if b.kind != pageOpener && b.kind != pageFront {
		return
	}
	if b.kind == pageFront && b.pdf.PageNo() == 1 {
		return
	}
	left, right := b.margins()
	num := b.folio(b.pdf.PageNo())
	b.pdf.SetTextColor(90, 90, 90)
	b.font(b.sans, "", 8)
	w := b.pdf.GetStringWidth(num)
	b.pdf.Text(left+(b.pageW-left-right-w)/2, b.pageH-pdfMarginBottom+10, num)
	b.pdf.SetTextColor(0, 0, 0)
}

// inlineText collects the plain text below a node
func inlineText(n ast.Node) string {
	var sb strings.Builder
	ast.WalkFunc(n, func(node ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.GoToNext
		}
		switch v := node.(type) {
		case *ast.Text:
			sb.Write(v.Literal)
		case *ast.Code:
			sb.Write(v.Literal)
		case *ast.Softbreak, *ast.Hardbreak:
			sb.WriteByte(' ')
		}
		return ast.GoToNext
	})
	return strings.Join(strings.Fields(sb.String()), " ")
}

// words turns the inline content of a node into words made of styled runs
func (b *bookPDF) words(n ast.Node, baseStyle string, size float64) []bookWord {
	var (
		out  []bookWord
		cur  bookWord
		bold = strings.Contains(baseStyle, "B")
		ital = strings.Contains(baseStyle, "I")
	)
	style := func(mono bool) string {
		if mono {
			return "M"
		}
		s := ""
		if bold {
			s += "B"
		}
		if ital {
			s += "I"
		}
		return s
	}
	flush := func() {
		if len(cur.runs) > 0 {
			out = append(out, cur)
		}
		cur = bookWord{}
	}
	add := func(text string, mono bool) {
		st := style(mono)
		for _, r := range text {
			if unicode.IsSpace(r) {
				flush()
				continue
			}
			if k := len(cur.runs) - 1; k >= 0 && cur.runs[k].style == st {
				cur.runs[k].text += string(r)
			} else {
				cur.runs = append(cur.runs, bookRun{string(r), st, size})
			}
		}
	}
	var walk func(node ast.Node)
	walk = func(node ast.Node) {
		switch v := node.(type) {
		case *ast.Text:
			add(string(v.Literal), false)
		case *ast.Code:
			add(string(v.Literal), true)
		case *ast.Softbreak:
			flush()
		case *ast.Hardbreak:
			flush()
			if len(out) > 0 {
				out[len(out)-1].breakLine = true
			}
		case *ast.HTMLSpan:
		case *ast.Strong:
			old := bold
			bold = true
			for _, c := range node.GetChildren() {
				walk(c)
			}
			bold = old
		case *ast.Emph:
			old := ital
			ital = !ital
			for _, c := range node.GetChildren() {
				walk(c)
			}
			ital = old
		default:
			for _, c := range node.GetChildren() {
				walk(c)
			}
		}
	}
	for _, c := range n.GetChildren() {
		walk(c)
	}
	flush()
	for i := range out {
		for _, r := range out[i].runs {
			out[i].width += b.runWidth(r)
		}
	}
	return out
}

// paragraph sets words in lines between the given indentations, justified unless ragged is set
func (b *bookPDF) paragraph(ws []bookWord, size, lineH, indent float64, ragged bool) {
	b.setRunFont("", size)
	space := b.pdf.GetStringWidth(" ")
	for len(ws) > 0 {
		b.ensure(lineH)
		left, _ := b.margins()
		x0 := left + indent
		avail := b.measure() - indent
		n, used := 0, 0.0
		for n < len(ws) {
			w := ws[n].width
			if n > 0 {
				w += space
			}
			if n > 0 && used+w > avail {
				break
			}
			used += w
			n++
			if ws[n-1].breakLine {
				break
			}
		}
		line := ws[:n]
		ws = ws[n:]
		last := len(ws) == 0 || line[len(line)-1].breakLine
		gap := space
		if !ragged && !last && len(line) > 1 {
			words := used - space*float64(len(line)-1)
			if g := (avail - words) / float64(len(line)-1); g < space*3 {
				gap = g
			}
		}
		b.quoteBar(b.y, lineH)
		x, base := x0, b.y+lineH*0.74
		for _, w := range line {
			for _, r := range w.runs {
				b.setRunFont(r.style, r.size)
				t := b.tr(r.text)
				rw := b.pdf.GetStringWidth(t)
				if r.style == "M" {
					b.pdf.SetFillColor(236, 236, 236)
					b.pdf.Rect(x-0.4, base-r.size*0.28, rw+0.8, r.size*0.38, "F")
				}
				b.pdf.Text(x, base, t)
				x += rw
			}
			x += gap
		}
		b.y += lineH
	}
}

// wrapText breaks plain text into lines that fit the width
func (b *bookPDF) wrapText(family, style string, size float64, text string, width float64) []string {
	b.font(family, style, size)
	var lines []string
	cur := ""
	for w := range strings.FieldsSeq(text) {
		try := w
		if cur != "" {
			try = cur + " " + w
		}
		if cur != "" && b.pdf.GetStringWidth(b.tr(try)) > width {
			lines = append(lines, cur)
			cur = w
		} else {
			cur = try
		}
	}
	if cur != "" || len(lines) == 0 {
		lines = append(lines, cur)
	}
	return lines
}

// chapterParts splits a chapter heading into its number and title ("Chapter 3: Tools" or just "Tools")
func (b *bookPDF) chapterParts(text string, next int) (int, string) {
	if m := chapterLabel.FindStringSubmatch(text); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n, strings.TrimSpace(m[2])
	}
	return next, text
}

// outline collects the table of contents, with the same numbering as the chapters and sections get
func (b *bookPDF) outline(nodes []ast.Node, chapterLevel int) []tocEntry {
	var out []tocEntry
	chap, sec := 0, 0
	for _, n := range nodes {
		h, ok := n.(*ast.Heading)
		if !ok {
			continue
		}
		text := inlineText(h)
		switch {
		case h.Level <= chapterLevel:
			num, title := b.chapterParts(text, chap+1)
			chap, sec = num, 0
			if title == "" {
				title = text
			}
			out = append(out, tocEntry{0, strconv.Itoa(num), title})
		case h.Level == chapterLevel+1 && chap > 0:
			sec++
			out = append(out, tocEntry{1, fmt.Sprintf("%d.%d", chap, sec), text})
		}
	}
	return out
}

func (b *bookPDF) titlePage(title string, lines []string) {
	b.addPage(pageTitle)
	left, _ := b.margins()
	w := b.measure()
	b.y = b.pageH * 0.30
	for _, l := range b.wrapText(b.sans, "B", 28, title, w) {
		b.font(b.sans, "B", 28)
		b.pdf.Text(left, b.y, b.tr(l))
		b.y += 12
	}
	b.pdf.SetFillColor(30, 30, 30)
	b.pdf.Rect(left, b.y-4, w, 1.2, "F")
	b.y += 6
	for _, l := range lines {
		for _, s := range b.wrapText(b.serif, "I", 13, l, w) {
			b.font(b.serif, "I", 13)
			b.pdf.Text(left, b.y, b.tr(s))
			b.y += 6.5
		}
	}
}

func (b *bookPDF) contentsPages() {
	if len(b.toc) == 0 {
		return
	}
	b.addPage(pageFront)
	left, _ := b.margins()
	w := b.measure()
	b.y = pdfMarginTop + 14
	b.font(b.sans, "B", 22)
	b.pdf.Text(left, b.y, b.tr("Contents"))
	b.pdf.SetFillColor(30, 30, 30)
	b.pdf.Rect(left, b.y+3.5, w, 0.6, "F")
	b.y += 14
	for i, e := range b.toc {
		page := "00"
		if b.tocPages != nil {
			page = b.folio(b.tocPages[i])
		}
		size, family, style, indent, h := 9.5, b.serif, "", 9.0, 5.2
		if e.level == 0 {
			size, family, style, indent, h = 10.5, b.sans, "B", 0, 6.4
			if i > 0 {
				b.y += 2
			}
		}
		if b.y+h > b.bottom() {
			b.addPage(pageFront)
			b.y = pdfMarginTop + 6
			left, _ = b.margins()
		}
		label := e.num + "  " + e.title
		b.font(family, style, size)
		lw := b.pdf.GetStringWidth(b.tr(label))
		pw := b.pdf.GetStringWidth(page)
		maxW := w - indent - pw - 6
		for lw > maxW && len(label) > 4 {
			label = strings.TrimSuffix(label[:len(label)-4], " ") + "…"
			lw = b.pdf.GetStringWidth(b.tr(label))
		}
		base := b.y + h*0.7
		b.pdf.Text(left+indent, base, b.tr(label))
		b.pdf.Text(left+w-pw, base, page)
		if e.level == 1 {
			b.pdf.SetTextColor(150, 150, 150)
			dot := b.pdf.GetStringWidth(" .")
			for x := left + indent + lw + 2; x+dot < left+w-pw-2; x += dot {
				b.pdf.Text(x, base, " .")
			}
			b.pdf.SetTextColor(0, 0, 0)
		}
		b.y += h
	}
}

func (b *bookPDF) chapterOpener(text string) {
	num, title := b.chapterParts(text, b.chapNum+1)
	b.chapNum, b.secNum, b.figNum = num, 0, 0
	if b.mainStart == 0 {
		b.mainStart = b.pdf.PageNo() + 1
	}
	b.addPage(pageOpener)
	left, _ := b.margins()
	w := b.measure()
	b.y = pdfMarginTop + 22
	label := fmt.Sprintf("CHAPTER %d", num)
	if title == "" {
		title, label = text, ""
	}
	b.chapHead = fmt.Sprintf("Chapter %d: %s", num, title)
	if strings.EqualFold(title, fmt.Sprintf("Chapter %d", num)) {
		b.chapHead = title
	}
	b.secHead = b.chapHead
	if b.tocAt < len(b.toc) {
		b.tocAt++
	}
	b.pdf.Bookmark(b.tr(b.chapHead), 0, -1)
	if label != "" {
		b.pdf.SetTextColor(120, 120, 120)
		b.font(b.sans, "B", 9.5)
		b.pdf.Text(left, b.y, label)
		b.pdf.SetTextColor(0, 0, 0)
		b.y += 11
	}
	for _, l := range b.wrapText(b.sans, "B", 24, title, w) {
		b.font(b.sans, "B", 24)
		b.pdf.Text(left, b.y, b.tr(l))
		b.y += 10.5
	}
	b.pdf.SetFillColor(30, 30, 30)
	b.pdf.Rect(left, b.y-4, w, 1.0, "F")
	b.y += 14
}

func (b *bookPDF) section(text string, level int) {
	numbered := level == 1 && b.chapNum > 0
	label := text
	size, before, after := 12.5, 6.0, 2.5
	if numbered {
		b.secNum++
		label = fmt.Sprintf("%d.%d  %s", b.chapNum, b.secNum, text)
		b.secHead = label
		if b.tocAt < len(b.toc) {
			b.tocAt++
		}
	} else {
		size, before, after = 10.8, 4.5, 1.8
	}
	lines := b.wrapText(b.sans, "B", size, label, b.measure())
	if b.y > pdfMarginTop+1 {
		b.space(before)
	}
	b.ensure(float64(len(lines))*size*0.5 + 2*pdfLineHeight)
	if numbered {
		b.pdf.Bookmark(b.tr(label), 1, -1)
	}
	left, _ := b.margins()
	for _, l := range lines {
		b.font(b.sans, "B", size)
		b.quoteBar(b.y, size*0.5)
		b.pdf.Text(left, b.y+size*0.38, b.tr(l))
		b.y += size * 0.5
	}
	b.y += after
}

func (b *bookPDF) image(dest, alt string) {
	path := dest
	if !filepath.IsAbs(path) {
		path = filepath.Join(b.dir, path)
	}
	info := b.pdf.RegisterImageOptions(path, gofpdf.ImageOptions{ReadDpi: true})
	if info == nil || b.pdf.Err() {
		b.pdf.ClearError()
		return
	}
	left, _ := b.margins()
	maxW := b.measure()
	w, h := info.Width(), info.Height()
	if w > maxW {
		h, w = h*maxW/w, maxW
	}
	if limit := (b.pageH - pdfMarginTop - pdfMarginBottom) * 0.6; h > limit {
		w, h = w*limit/h, limit
	}
	b.ensure(h + 12)
	b.pdf.ImageOptions(path, left+(maxW-w)/2, b.y+2, w, h, false, gofpdf.ImageOptions{ReadDpi: true}, 0, "")
	b.y += h + 4
	caption := alt
	if b.chapNum > 0 {
		b.figNum++
		caption = strings.TrimSpace(fmt.Sprintf("Figure %d-%d. %s", b.chapNum, b.figNum, alt))
	}
	if caption != "" {
		for _, l := range b.wrapText(b.sans, "", 8.5, caption, maxW) {
			b.font(b.sans, "", 8.5)
			lw := b.pdf.GetStringWidth(b.tr(l))
			b.pdf.Text(left+(maxW-lw)/2, b.y+3, b.tr(l))
			b.y += 4.2
		}
	}
	b.y += 4
}

func (b *bookPDF) codeBlock(code string, indent float64) {
	left, _ := b.margins()
	x := left + indent
	w := b.measure() - indent
	b.font(b.mono, "", pdfCodeSize)
	cw := b.pdf.GetStringWidth("M")
	cols := max(int((w-7)/cw), 20)
	var lines []string
	for l := range strings.SplitSeq(strings.TrimRight(code, "\n"), "\n") {
		r := []rune(strings.ReplaceAll(l, "\t", "    "))
		for len(r) > cols {
			lines = append(lines, string(r[:cols]))
			r = append([]rune("↪ "), r[cols:]...)
		}
		lines = append(lines, string(r))
	}
	total := float64(len(lines))*pdfCodeLine + 5
	if total < b.pageH-pdfMarginTop-pdfMarginBottom {
		b.ensure(total)
	}
	segment := func(h float64) {
		b.pdf.SetFillColor(244, 244, 244)
		b.pdf.Rect(x, b.y, w, h, "F")
		b.pdf.SetFillColor(120, 120, 120)
		b.pdf.Rect(x, b.y, 0.8, h, "F")
	}
	b.space(1.5)
	segment(2.5)
	b.y += 2.5
	for _, l := range lines {
		if b.y+pdfCodeLine > b.bottom() {
			b.addPage(pageBody)
			left, _ = b.margins()
			x = left + indent
		}
		segment(pdfCodeLine)
		b.font(b.mono, "", pdfCodeSize)
		b.pdf.Text(x+4, b.y+pdfCodeLine*0.72, b.tr(l))
		b.y += pdfCodeLine
	}
	segment(2.5)
	b.y += 2.5
	b.space(pdfParSpace + 1)
}

func (b *bookPDF) table(t *ast.Table) {
	var rows [][]string
	var header []bool
	ast.WalkFunc(t, func(node ast.Node, entering bool) ast.WalkStatus {
		if row, ok := node.(*ast.TableRow); ok && entering {
			var cells []string
			isHead := false
			for _, c := range row.Children {
				if cell, ok := c.(*ast.TableCell); ok {
					cells = append(cells, inlineText(cell))
					isHead = isHead || cell.IsHeader
				}
			}
			rows = append(rows, cells)
			header = append(header, isHead)
			return ast.SkipChildren
		}
		return ast.GoToNext
	})
	if len(rows) == 0 {
		return
	}
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r))
	}
	left, _ := b.margins()
	w := b.measure()
	const size, lh, pad = 8.5, 4.0, 1.6
	widths := make([]float64, cols)
	for _, r := range rows {
		for i, c := range r {
			widths[i] = max(widths[i], b.textWidth(b.sans, "B", size, c)+2*pad+1)
		}
	}
	sum := 0.0
	for _, cw := range widths {
		sum += cw
	}
	if sum > w {
		for i := range widths {
			widths[i] = max(widths[i]*w/sum, 12)
		}
	}
	b.space(2)
	b.ensure(lh*3 + 4)
	b.pdf.SetDrawColor(30, 30, 30)
	b.pdf.SetLineWidth(0.4)
	b.pdf.Line(left, b.y, left+min(sum, w), b.y)
	for ri, r := range rows {
		style := ""
		if header[ri] {
			style = "B"
		}
		cells := make([][]string, cols)
		n := 1
		for i := 0; i < cols; i++ {
			text := ""
			if i < len(r) {
				text = r[i]
			}
			cells[i] = b.wrapText(b.sans, style, size, text, widths[i]-2*pad)
			n = max(n, len(cells[i]))
		}
		h := float64(n)*lh + 2*pad
		if b.y+h > b.bottom() {
			b.addPage(pageBody)
			left, _ = b.margins()
		}
		x := left
		for i := 0; i < cols; i++ {
			for li, l := range cells[i] {
				b.font(b.sans, style, size)
				b.pdf.Text(x+pad, b.y+pad+float64(li)*lh+lh*0.75, b.tr(l))
			}
			x += widths[i]
		}
		b.y += h
		if header[ri] {
			b.pdf.SetLineWidth(0.2)
			b.pdf.Line(left, b.y, left+min(sum, w), b.y)
		}
	}
	b.pdf.SetLineWidth(0.4)
	b.pdf.Line(left, b.y, left+min(sum, w), b.y)
	b.y += 2
	b.space(pdfParSpace + 1)
}

func (b *bookPDF) block(n ast.Node, indent float64, italic bool, chapterLevel int) {
	style := ""
	if italic {
		style = "I"
	}
	switch v := n.(type) {
	case *ast.Heading:
		text := inlineText(v)
		if v.Level <= chapterLevel {
			b.chapterOpener(text)
			return
		}
		level := v.Level - chapterLevel
		b.section(text, level)
	case *ast.Paragraph:
		if len(v.Children) == 1 {
			if img, ok := v.Children[0].(*ast.Image); ok {
				b.image(string(img.Destination), inlineText(img))
				return
			}
		}
		b.paragraph(b.words(v, style, pdfBodySize), pdfBodySize, pdfLineHeight, indent, false)
		b.space(pdfParSpace)
	case *ast.BlockQuote:
		b.quote++
		for _, c := range v.Children {
			b.block(c, indent+5, true, chapterLevel)
		}
		b.quote--
		b.space(1)
	case *ast.List:
		num := v.Start
		if num == 0 {
			num = 1
		}
		for _, item := range v.Children {
			mark := "•"
			if v.ListFlags&ast.ListTypeOrdered != 0 {
				mark = strconv.Itoa(num) + "."
				num++
			}
			b.ensure(pdfLineHeight)
			left, _ := b.margins()
			b.setRunFont(style, pdfBodySize)
			mw := b.pdf.GetStringWidth(b.tr(mark))
			b.pdf.Text(left+indent+5-mw, b.y+pdfLineHeight*0.74, b.tr(mark))
			for _, c := range item.GetChildren() {
				switch p := c.(type) {
				case *ast.Paragraph:
					b.paragraph(b.words(p, style, pdfBodySize), pdfBodySize, pdfLineHeight, indent+7, false)
				default:
					b.block(c, indent+7, italic, chapterLevel)
				}
			}
			b.space(0.8)
		}
		b.space(pdfParSpace - 0.8)
	case *ast.CodeBlock:
		b.codeBlock(string(v.Literal), indent)
	case *ast.Table:
		b.table(v)
	case *ast.HorizontalRule:
		b.space(2)
		b.ensure(4)
		left, _ := b.margins()
		mid := left + b.measure()/2
		b.pdf.SetDrawColor(140, 140, 140)
		b.pdf.SetLineWidth(0.2)
		b.pdf.Line(mid-15, b.y+1, mid+15, b.y+1)
		b.space(5)
	case *ast.HTMLBlock:
	default:
		for _, c := range n.GetChildren() {
			b.block(c, indent, italic, chapterLevel)
		}
	}
}

// render lays out the whole book, with the given page numbers in the table of contents
func renderBook(doc ast.Node, sourceFilename string, tocPages []int, mainStart int) (*bookPDF, []int, error) {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{UnitStr: "mm", Size: gofpdf.SizeType{Wd: pdfPageW, Ht: pdfPageH}, OrientationStr: "P"})
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetMargins(0, 0, 0)
	b := &bookPDF{pdf: pdf, dir: filepath.Dir(sourceFilename), tocPages: tocPages, mainStart: mainStart}
	b.pageW, b.pageH = pdf.GetPageSize()
	b.loadFonts()
	pdf.SetHeaderFuncMode(b.header, false)
	pdf.SetFooterFunc(b.footer)

	children := doc.GetChildren()
	title := strings.TrimSuffix(filepath.Base(sourceFilename), filepath.Ext(sourceFilename))
	var titleLines []string
	i := 0
	if len(children) > 0 {
		if h, ok := children[0].(*ast.Heading); ok && h.Level == 1 {
			title = inlineText(h)
			for i = 1; i < len(children); i++ {
				if p, ok := children[i].(*ast.Paragraph); ok {
					titleLines = append(titleLines, inlineText(p))
					continue
				}
				if _, ok := children[i].(*ast.HTMLBlock); ok {
					continue
				}
				break
			}
		}
	}
	// Chapters are the level 2 headings below a level 1 title, or the level 1 headings otherwise
	chapterLevel := 1
	if i > 0 {
		chapterLevel = 2
	}
	b.title = title
	b.chapHead, b.secHead = title, title
	pdf.SetTitle(title, true)
	pdf.SetCreator("Orbiton", true)
	b.toc = b.outline(children[i:], chapterLevel)
	b.titlePage(title, titleLines)
	b.contentsPages()
	var pages []int
	started := false
	for ; i < len(children); i++ {
		if _, ok := children[i].(*ast.HTMLBlock); ok {
			continue
		}
		if !started {
			started = true
			if h, ok := children[i].(*ast.Heading); !ok || h.Level > chapterLevel {
				if b.mainStart == 0 {
					b.mainStart = b.pdf.PageNo() + 1
				}
				b.addPage(pageBody)
			}
		}
		before := b.tocAt
		b.block(children[i], 0, false, chapterLevel)
		for ; before < b.tocAt; before++ {
			pages = append(pages, b.pdf.PageNo())
		}
	}
	if err := pdf.Error(); err != nil {
		return nil, nil, err
	}
	return b, pages, nil
}

// ExportBookPDF typesets a Markdown book as a PDF that looks like a technical book: a title page,
// a table of contents, numbered chapters and sections, running heads, code listings, tables and figures.
// The book is laid out twice, so that the table of contents gets the right page numbers.
func ExportBookPDF(markdownSource []byte, sourceFilename, pdfFilename string) error {
	parse := func() ast.Node {
		return parser.NewWithExtensions(parser.CommonExtensions).Parse(markdownSource)
	}
	first, pages, err := renderBook(parse(), sourceFilename, nil, 0)
	if err != nil {
		return fmt.Errorf("could not export %s: %w", filepath.Base(pdfFilename), err)
	}
	b, _, err := renderBook(parse(), sourceFilename, pages, first.mainStart)
	if err != nil {
		return fmt.Errorf("could not export %s: %w", filepath.Base(pdfFilename), err)
	}
	return b.pdf.OutputFileAndClose(pdfFilename)
}

// bookPDFViewerRunning is true while a PDF viewer started by viewBookPDF is open.
// Zathura reloads a PDF when it changes, so there is no need to start another one.
var bookPDFViewerRunning atomic.Bool

// viewBookPDF saves the book, exports it as a PDF next to it and shows it in Zathura,
// or in the browser if Zathura is not installed.
func (e *Editor) viewBookPDF(c *vt.Canvas, tty *vt.TTY, status *StatusBar) {
	status.ClearAll(c, false)
	if e.changed.Load() {
		if err := e.Save(c, tty); err != nil {
			status.SetError(err)
			status.Show(c, e)
			return
		}
	}
	if files.WhichCached("zathura") == "" {
		e.viewMarkdownInBrowser(c, tty, status)
		return
	}
	absFilename, err := filepath.Abs(e.filename)
	if err != nil {
		status.SetError(err)
		status.Show(c, e)
		return
	}
	data, err := os.ReadFile(absFilename)
	if err != nil {
		status.SetError(err)
		status.Show(c, e)
		return
	}
	pdfFilename := strings.TrimSuffix(absFilename, filepath.Ext(absFilename)) + ".pdf"
	if err := ExportBookPDF(data, absFilename, pdfFilename); err != nil {
		status.SetError(err)
		status.Show(c, e)
		return
	}
	if bookPDFViewerRunning.Load() {
		status.SetMessage("Updated " + filepath.Base(pdfFilename))
		status.Show(c, e)
		return
	}
	cmd := exec.Command("zathura", pdfFilename)
	cmd.Dir = filepath.Dir(absFilename)
	if err := cmd.Start(); err != nil {
		status.SetError(err)
		status.Show(c, e)
		return
	}
	bookPDFViewerRunning.Store(true)
	go func() {
		_ = cmd.Wait()
		bookPDFViewerRunning.Store(false)
	}()
	status.SetMessage("Opening " + filepath.Base(pdfFilename) + " in Zathura")
	status.Show(c, e)
}
