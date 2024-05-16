package palgen

import (
	"errors"
	"image"
	"image/color"
	"math"
	"sort"
)

// SortablePalette is a slice of color.Color that can be sorted with sort.Sort, by euclidian distance for R, G and B
type SortablePalette []color.Color

// Length from RGB (0, 0, 0)
func colorLength(c color.Color) float64 {
	r := c.(color.RGBA)
	return math.Sqrt(float64(r.R*r.R + r.G*r.G + r.B*r.B)) // + r.A*r.A))
}

func (a SortablePalette) Len() int           { return len(a) }
func (a SortablePalette) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortablePalette) Less(i, j int) bool { return colorLength(a[i]) < colorLength(a[j]) }

// Median finds not the average but the median color
func Median(colors []color.Color) (color.Color, error) {
	length := len(colors)
	if length == 0 {
		return nil, errors.New("can't find the median of an empty slice of colors")
	}
	if len(colors) == 1 {
		return colors[0], nil
	}

	// 1. Sort the colors
	sp := SortablePalette(colors)
	sort.Sort(sp)

	// 2. Select the center one, if odd
	if length%2 != 0 {
		centerPos := length / 2
		return sp[centerPos], nil
	}
	// 3. If the numbers are even, select the two center one and take the average of those
	centerPos1 := (length / 2) - 1
	centerPos2 := length / 2
	c1 := sp[centerPos1].(color.RGBA)
	c2 := sp[centerPos2].(color.RGBA)
	r := (c1.R + c2.R) / 2.0
	g := (c1.G + c2.G) / 2.0
	b := (c1.B + c2.B) / 2.0
	a := (c1.A + c2.A) / 2.0
	// return the new color
	return color.RGBA{r, g, b, a}, nil
}

// Median3 finds not the average but the median color. Returns three colors if the number of colors is even (average, first and second color in the center).
func Median3(colors []color.Color) (color.Color, color.Color, color.Color, error) {
	length := len(colors)
	if length == 0 {
		return nil, nil, nil, errors.New("can't find the median of an empty slice of colors")
	}
	if len(colors) == 1 {
		return colors[0], colors[0], colors[0], nil
	}

	// 1. Sort the colors
	sp := SortablePalette(colors)
	sort.Sort(sp)

	// 2. Select the center one, if odd
	if length%2 != 0 {
		centerPos := length / 2
		// Return the center color, thrice
		return sp[centerPos], sp[centerPos], sp[centerPos], nil
	}
	// 3. If the numbers are even, select the two center one and take the average of those
	centerPos1 := (length / 2) - 1
	centerPos2 := length / 2
	c1 := sp[centerPos1].(color.RGBA)
	c2 := sp[centerPos2].(color.RGBA)
	r := (c1.R + c2.R) / 2.0
	g := (c1.G + c2.G) / 2.0
	b := (c1.B + c2.B) / 2.0
	a := (c1.A + c2.A) / 2.0
	averageColor := color.RGBA{r, g, b, a}

	// Also return the two center colors
	return averageColor, sp[centerPos2], sp[centerPos1], nil
}

// Generate can generate a palette with N colors, given an image
func Generate(img image.Image, N int) (color.Palette, error) {
	groups := make(map[int][]color.Color)
	already := make(map[color.Color]bool)

	numberOfPixels := (img.Bounds().Max.Y - img.Bounds().Min.Y) * (img.Bounds().Max.X - img.Bounds().Min.X)
	levelCounter := make(map[int]float64, numberOfPixels)
	colorCounter := make(map[color.RGBA]float64, numberOfPixels)

	// Pick out the colors from the image, per intensity level, and store them in the groups map
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := img.At(x, y)
			gc := color.GrayModel.Convert(c).(color.Gray)
			rgba := color.RGBAModel.Convert(c).(color.RGBA)
			level := int(float64(gc.Y) / (255.0 / float64(N-1)))
			alreadyColor, ok := already[rgba]
			if !alreadyColor || !ok {
				groups[level] = append(groups[level], rgba)
				already[rgba] = true
			}
			// Count the different levels used
			if counter, ok := levelCounter[level]; ok {
				levelCounter[level] = counter + 1.0
			} else {
				levelCounter[level] = 1.0
			}
			// Count the different colors used
			if counter, ok := colorCounter[rgba]; ok {
				colorCounter[rgba] = counter + 1.0
			} else {
				colorCounter[rgba] = 1.0
			}

		}
	}

	// Reset the map for if colors are already appended to a slice
	already = make(map[color.Color]bool)
	already2 := make(map[color.Color]bool)
	var extrapal color.Palette

	// Remove the group of colors with the least used intensity level
	minCoverage := 1.0
	minCoverageKey := 0
	found := false
	for intensityLevelKey := range groups {
		coverage := levelCounter[intensityLevelKey] / float64(numberOfPixels)
		if coverage < minCoverage {
			minCoverage = coverage
			minCoverageKey = intensityLevelKey
			found = true
		}
	}
	if found {
		delete(groups, minCoverageKey)
	}

	// Find the median color for each intensity level
	var pal color.Palette
	for _, colors := range groups {
		// Find the median color of a group of colors of a certain intensity
		//medianColor, err := Median(colors)
		medianColor1, medianColor2, medianColor3, err := Median3(colors)
		if err != nil {
			return nil, err
		}
		// Add the medianColor1 to the palette, if it's not already there
		alreadyColor, ok := already[medianColor1]
		if !alreadyColor || !ok {
			pal = append(pal, medianColor1)
			already[medianColor1] = true
		}
		// Add medianColor2 and medianColor3 to the extra palette, if they are not already in it
		alreadyColor2, ok := already2[medianColor2]
		if !alreadyColor2 || !ok {
			extrapal = append(extrapal, medianColor2)
			already2[medianColor2] = true
		}
		alreadyColor2, ok = already2[medianColor3]
		if !alreadyColor2 || !ok {
			extrapal = append(extrapal, medianColor3)
			already2[medianColor3] = true
		}
	}

	// If there are not enough colors in the generated palette (N+1, because we will remove one), add colors from extrapal
	for (len(pal) < N) && (len(extrapal) > 0) {
		var (
			maxUsage          = -1.0
			mostFrequentIndex = -1
		)

		// Loop through extrapal to find the most frequently used color
		for i, extraColor := range extrapal {
			rgbaExtra := color.RGBAModel.Convert(extraColor).(color.RGBA)
			if usage, ok := colorCounter[rgbaExtra]; ok {
				if usage > maxUsage {
					maxUsage = usage
					mostFrequentIndex = i
				}
			}
		}
		if mostFrequentIndex == -1 { // not found
			break
		}

		// Use the most frequently used color from the palette of extra colors
		c := extrapal[mostFrequentIndex]

		// Add the color to the palette, if it's not already there
		alreadyColor, ok := already[c]
		if !alreadyColor || !ok {
			pal = append(pal, c)
			already[c] = true
		}

		// Remove this color from extrapal
		extrapal = append(extrapal[:mostFrequentIndex], extrapal[mostFrequentIndex+1:]...)
	}

	// Now remove the one extra color that covers the least amount of pixels of the image that has been converted to the current palette
	if len(pal) > N {
		minCoverage := 1.0
		minCoverageIndex := -1
		for i := 0; i < len(pal); i++ {
			c := color.RGBAModel.Convert(pal[i]).(color.RGBA)
			coverage := colorCounter[c] / float64(numberOfPixels)
			if coverage < minCoverage {
				minCoverage = coverage
				minCoverageIndex = i
			}
		}
		if minCoverageIndex != -1 {
			// Replace i with the last element
			pal[minCoverageIndex] = pal[len(pal)-1]
			// Remove the last element
			pal = pal[:len(pal)-1]
		}
	}

	// Return the generated palette
	return pal, nil
}

// GenerateUpTo can generate a palette with up to N colors, given an image
func GenerateUpTo(img image.Image, N int) (color.Palette, error) {
	groups := make(map[int][]color.Color)
	already := make(map[color.Color]bool)

	numberOfPixels := (img.Bounds().Max.Y - img.Bounds().Min.Y) * (img.Bounds().Max.X - img.Bounds().Min.X)
	levelCounter := make(map[int]float64, numberOfPixels)
	colorCounter := make(map[color.RGBA]float64, numberOfPixels)

	// Pick out the colors from the image, per intensity level, and store them in the groups map
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := img.At(x, y)
			gc := color.GrayModel.Convert(c).(color.Gray)
			rgba := color.RGBAModel.Convert(c).(color.RGBA)
			level := int(float64(gc.Y) / (255.0 / float64(N-1)))
			alreadyColor, ok := already[rgba]
			if !alreadyColor || !ok {
				groups[level] = append(groups[level], rgba)
				already[rgba] = true
			}
			// Count the different levels used
			if counter, ok := levelCounter[level]; ok {
				levelCounter[level] = counter + 1.0
			} else {
				levelCounter[level] = 1.0
			}
			// Count the different colors used
			if counter, ok := colorCounter[rgba]; ok {
				colorCounter[rgba] = counter + 1.0
			} else {
				colorCounter[rgba] = 1.0
			}
		}
	}

	// Remove the group of colors with the least used intensity level
	minCoverage := 1.0
	minCoverageKey := 0
	found := false
	for intensityLevelKey := range groups {
		coverage := levelCounter[intensityLevelKey] / float64(numberOfPixels)
		if coverage < minCoverage {
			minCoverage = coverage
			minCoverageKey = intensityLevelKey
			found = true
		}
	}
	if found {
		delete(groups, minCoverageKey)
	}

	// Reset the map for if colors are already appended to a slice
	already = make(map[color.Color]bool)

	// Find the median color for each intensity level
	var pal color.Palette
	for _, colors := range groups {
		// Find the median color of a group of colors of a certain intensity
		//medianColor, err := Median(colors)
		medianColor1, _, _, err := Median3(colors)
		if err != nil {
			return nil, err
		}
		// Add the medianColor1 to the palette, if it's not already there
		alreadyColor, ok := already[medianColor1]
		if !alreadyColor || !ok {
			pal = append(pal, medianColor1)
			already[medianColor1] = true
		}
	}

	// Return the generated palette
	return pal, nil
}
