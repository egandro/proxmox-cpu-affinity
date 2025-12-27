package svg

import "fmt"

type rgba struct {
	r, g, b, a float64
}

func (c rgba) String() string {
	return fmt.Sprintf("rgb(%d,%d,%d)", int(c.r), int(c.g), int(c.b))
}

func calculateCellColor(t float64) (rgba, string) {
	// RdYlBu_r Colormap (Blue -> Yellow -> Red)
	// Approximated from Matplotlib's RdYlBu_r

	stops := []struct {
		val float64
		col rgba
	}{
		{0.00, rgba{49, 54, 149, 1.0}},   // Deep Blue
		{0.25, rgba{116, 173, 209, 1.0}}, // Light Blue
		{0.50, rgba{255, 255, 191, 1.0}}, // Pale Yellow
		{0.75, rgba{253, 174, 97, 1.0}},  // Orange
		{1.00, rgba{215, 48, 39, 1.0}},   // Red
	}

	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	var c1, c2 rgba
	var t1, t2 float64

	// Find the segment t falls into
	for i := 0; i < len(stops)-1; i++ {
		if t >= stops[i].val && t <= stops[i+1].val {
			c1 = stops[i].col
			c2 = stops[i+1].col
			t1 = stops[i].val
			t2 = stops[i+1].val
			break
		}
	}

	// Linear interpolation
	f := (t - t1) / (t2 - t1)
	r := c1.r + f*(c2.r-c1.r)
	g := c1.g + f*(c2.g-c1.g)
	b := c1.b + f*(c2.b-c1.b)
	a := c1.a + f*(c2.a-c1.a)

	// Determine text color based on luminance
	// Standard luminance formula: 0.299R + 0.587G + 0.114B
	lum := 0.299*r + 0.587*g + 0.114*b
	textColor := "black"
	if lum < 128 {
		textColor = "white"
	}

	return rgba{r, g, b, a}, textColor
}
