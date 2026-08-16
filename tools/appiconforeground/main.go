package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: appiconforeground INPUT.png OUTPUT.png")
		os.Exit(2)
	}

	source, err := readPNG(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	foreground, err := extractForeground(source)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := writePNG(os.Args[2], foreground); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func readPNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open source icon: %w", err)
	}
	defer file.Close()

	value, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode source icon: %w", err)
	}
	return value, nil
}

func writePNG(path string, value image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create foreground icon: %w", err)
	}
	if err := png.Encode(file, value); err != nil {
		file.Close()
		return fmt.Errorf("encode foreground icon: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close foreground icon: %w", err)
	}
	return nil
}

// extractForeground keeps the central, closed llama artwork while removing
// the legacy icon's baked-in rounded rectangle and outer background. The
// darkest connected shape that does not touch the canvas edge is the llama's
// outline; flood filling around it also preserves its enclosed white areas.
func extractForeground(source image.Image) (*image.NRGBA, error) {
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return nil, fmt.Errorf("source icon is empty")
	}

	barriers := make([]bool, width*height)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
			barriers[indexOf(bounds, x, y)] = isArtworkPixel(pixel)
		}
	}

	component, err := interiorComponents(barriers, width, height)
	if err != nil {
		return nil, err
	}
	floodBarrier := dilate(component, width, height, 3)

	outside := make([]bool, len(component))
	queue := make([]image.Point, 0, 2*(width+height))
	addOutside := func(x, y int) {
		index := y*width + x
		if outside[index] || floodBarrier[index] {
			return
		}
		outside[index] = true
		queue = append(queue, image.Pt(x, y))
	}
	for x := 0; x < width; x++ {
		addOutside(x, 0)
		addOutside(x, height-1)
	}
	for y := 0; y < height; y++ {
		addOutside(0, y)
		addOutside(width-1, y)
	}

	directions := [...]image.Point{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}}
	for head := 0; head < len(queue); head++ {
		point := queue[head]
		for _, direction := range directions {
			next := point.Add(direction)
			if next.X >= 0 && next.X < width && next.Y >= 0 && next.Y < height {
				addOutside(next.X, next.Y)
			}
		}
	}

	inside := make([]bool, len(outside))
	for index := range inside {
		inside[index] = !outside[index]
	}
	inside = erode(inside, width, height, 3)
	for index, selected := range component {
		inside[index] = inside[index] || selected
	}

	output := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := y*width + x
			if !inside[index] {
				continue
			}
			pixel := color.NRGBAModel.Convert(source.At(x+bounds.Min.X, y+bounds.Min.Y)).(color.NRGBA)
			output.SetNRGBA(x, y, pixel)
		}
	}
	return output, nil
}

func isArtworkPixel(pixel color.NRGBA) bool {
	if pixel.A < 16 {
		return false
	}
	luminance := (2126*uint32(pixel.R) + 7152*uint32(pixel.G) + 722*uint32(pixel.B)) / 10000
	return luminance < 150
}

func interiorComponents(barriers []bool, width, height int) ([]bool, error) {
	visited := make([]bool, len(barriers))
	selected := make([]bool, len(barriers))
	selectedPixels := 0
	minimumArea := width * height / 10000
	directions := [...]image.Point{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}}

	for start, barrier := range barriers {
		if !barrier || visited[start] {
			continue
		}
		visited[start] = true
		queue := []int{start}
		touchesEdge := false
		for head := 0; head < len(queue); head++ {
			index := queue[head]
			x := index % width
			y := index / width
			if x == 0 || x == width-1 || y == 0 || y == height-1 {
				touchesEdge = true
			}
			for _, direction := range directions {
				nextX := x + direction.X
				nextY := y + direction.Y
				if nextX < 0 || nextX >= width || nextY < 0 || nextY >= height {
					continue
				}
				next := nextY*width + nextX
				if barriers[next] && !visited[next] {
					visited[next] = true
					queue = append(queue, next)
				}
			}
		}
		if !touchesEdge && len(queue) >= minimumArea {
			for _, index := range queue {
				selected[index] = true
			}
			selectedPixels += len(queue)
		}
	}

	if selectedPixels == 0 {
		return nil, fmt.Errorf("no enclosed foreground component found")
	}
	return selected, nil
}

func dilate(source []bool, width, height, radius int) []bool {
	result := make([]bool, len(source))
	for index, selected := range source {
		if !selected {
			continue
		}
		x := index % width
		y := index / width
		for offsetY := -radius; offsetY <= radius; offsetY++ {
			for offsetX := -radius; offsetX <= radius; offsetX++ {
				nextX := x + offsetX
				nextY := y + offsetY
				if nextX >= 0 && nextX < width && nextY >= 0 && nextY < height {
					result[nextY*width+nextX] = true
				}
			}
		}
	}
	return result
}

func erode(source []bool, width, height, radius int) []bool {
	result := make([]bool, len(source))
	for index, selected := range source {
		if !selected {
			continue
		}
		x := index % width
		y := index / width
		keep := true
		for offsetY := -radius; offsetY <= radius && keep; offsetY++ {
			for offsetX := -radius; offsetX <= radius; offsetX++ {
				nextX := x + offsetX
				nextY := y + offsetY
				if nextX < 0 || nextX >= width || nextY < 0 || nextY >= height || !source[nextY*width+nextX] {
					keep = false
					break
				}
			}
		}
		result[index] = keep
	}
	return result
}

func indexOf(bounds image.Rectangle, x, y int) int {
	return (y-bounds.Min.Y)*bounds.Dx() + x - bounds.Min.X
}
