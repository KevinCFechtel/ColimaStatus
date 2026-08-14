package tray

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
)

// The menu bar icon is a 2x crop of the official Colima logo. The original is
// distributed by the Colima project under the MIT license; see
// THIRD_PARTY_NOTICES.md.
const menuBarIconBase64 = "iVBORw0KGgoAAAANSUhEUgAAABoAAAAkCAYAAACXOioTAAAAAXNSR0IArs4c6QAAAERlWElmTU0AKgAAAAgAAYdpAAQAAAABAAAAGgAAAAAAA6ABAAMAAAABAAEAAKACAAQAAAABAAAAGqADAAQAAAABAAAAJAAAAABS2LDtAAABzWlUWHRYTUw6Y29tLmFkb2JlLnhtcAAAAAAAPHg6eG1wbWV0YSB4bWxuczp4PSJhZG9iZTpuczptZXRhLyIgeDp4bXB0az0iWE1QIENvcmUgNi4wLjAiPgogICA8cmRmOlJERiB4bWxuczpyZGY9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkvMDIvMjItcmRmLXN5bnRheC1ucyMiPgogICAgICA8cmRmOkRlc2NyaXB0aW9uIHJkZjphYm91dD0iIgogICAgICAgICAgICB4bWxuczpleGlmPSJodHRwOi8vbnMuYWRvYmUuY29tL2V4aWYvMS4wLyI+CiAgICAgICAgIDxleGlmOkNvbG9yU3BhY2U+NjU1MzU8L2V4aWY6Q29sb3JTcGFjZT4KICAgICAgICAgPGV4aWY6UGl4ZWxYRGltZW5zaW9uPjI2PC9leGlmOlBpeGVsWERpbWVuc2lvbj4KICAgICAgICAgPGV4aWY6UGl4ZWxZRGltZW5zaW9uPjM2PC9leGlmOlBpeGVsWURpbWVuc2lvbj4KICAgICAgPC9yZGY6RGVzY3JpcHRpb24+CiAgIDwvcmRmOlJERj4KPC94OnhtcG1ldGE+Cs0NUgUAAAbRSURBVEgNtVZpTFRXFP6AGYZN2UFGFNQKyObCaAwhTTC2KqIJaoLpDxrANNFEYzRx/2FiYuwfQ1M1cY8WrDQusaJxSdhEK8qm7DIjKhYcYGYQmGEb5vaea984A86UpulNZt595517vnvPPec7x4UxZgTgxX//5zC5OrOu1WpRXV3tTGXK35wC3b9/HytXrsTNmzenbNCRolMgnU6HoaEhHD58GB8+fHBkY0pyp0B0om3btiEgIABFRUVTMuhI6YtAAwMDuHDhAl68eIEtW7ZApVLh0aNHjmxMSS6z1eru7saVK1dw7tw5jI2NYd++fYiNjUVMTAyqqqrAIxQuLi62S6Y+p/DmP3bp0iU2e/ZsFhcXx06ePMn4/ZBYjNu3b7OEhARmNApVScz0egOrqa5l7e2vrTIHEyPt0vj48WOmVCrZmTNnGD/JJN07d+4wfjIBVF5eznbv3svWpm1icYlxTBkRzL7Pyp60ZoLAKFzX1taGadOmgZ8GZWVlwj3x8fEICQkRrhkeHoaXlxfc3d1x9OiP6B4uwdfpYUhRytHT6YvKGxqMjIxAoVA4dKUAWr58uTCUnZ0NNzc39PX1ITc3F0eOHBELKcynT58OmUyGhQkqNOtqMTRgRr9ehrdtRgyOadHVpUVk5GznQNHR0SguLobZbIaHhwe2b98u8kdaRSeWThcbF4PiX8YAZkGI0hMKD1fIPY1Qt2mcAlnD28/PD0FBQfDx8cH79+8xa9YsCQcdHR3cSKR4T0paBJPBB4uSfeEfLMdXcd4YHhpEQX4hfso7jZycH9Da2mJdK03swpuExG/t7e1ISkoSOqOjo1Cr1cjMzBTv7go5zCY5bpz/E6Y+X+h7dfDxs+B5yxV0W4rQWvcRy4pViI6OEfrS3ySghw8fwtvbGwsXLhQ6PT090Ov1OH/+As6euYS3nY1Q+Btg6BlDxppMVDwtxZqcfoRFeKOz3QhX2ThevmyW7FufdkDj4+O4ePEi1q1bJ6KQtBobG2EymfC85im+/c4TK3J9eUqEorpcgf5+I0ICw9DeqsWrlyb4BsjAzECbuhFmbkvGA0sa1jsiwZMnT1BfX4+srCzpO2pqapCSkoJvVqyFmcdAVakOr5tMGB0dQ0fnK0RGRGHIZEHQDAUWLPbFApU3dH3d6NMbrDZoYgd0+vRppKamIioqyqpEgUARFxkxFzrtkMix0HAPzAj3wuCIFjOVoeh96wIfXzfU/WFA2e86qF+pUVVdZbVhB0Q8R8SZk5Njp0B3VVdXxyMyAPouF6hS/WEZt8A0MA51qwb1DXVorjEhb68ahSe0kA/HQ5W0DFu3bkVhYeFnW0RBRBcPHjxg8+bN4/ylt2MPHgyM5xnz9/dncndXxu+BucnB3NxkjKcDW7JkMQsMDGR79uxh8+dHMY3mE++dPXuWhYaGMp6DZM9odR1lv6enp2CIz9sAOADCwsKwY8cOhIbMxIqv1+PXgt9QWloiknzDho3ghCsqcWBgAJTKMBw6dEiYIEorKCgQcysQMQKVBoo820GURGDEfSdO/Ay1pg23bt3i8xOiKJaWluLYsWOitKxevVrw4bt373Dw4EFRVu7evSvsCvams/F6wyIiIlhnZye92o38/HzGgVhTUxPj98UuX74sfvfu3WNUQjIyMtimTZvYx48fxbrKykrhar5Jxr3E+Kk+sTedgN+PIFReMsAX2R4KmzdvhkajATeI8PBwyOWcHTgvSu5OS0vDzp07RaLTQso7orMDBw6IFqCkpAQufAvWvm7//v2i4yF2sOU6CZVC/c2bN7BYLOIuyaXcCwJY0iHXb9y4UXBjXl6e2BDHMFldR2emCpqens7Wr1/PePdDon89eIownodStEnrP0cd7YiK26lTpwRbS5FDFHT16lXRbpG7aPDV4JsCvzPRxFBASINKCp107ty5kujTky+ybwS44NmzZyIwmpub2fHjxxmvrIy7UvQTnDkYZ3Y2Z84cIaMST20Ad5PY/fXr1xlPcsbvSbz//fc5GGzhly5dKrofCk26j1WrVomOqKWlBYODg+JOgoODRf5QneKRh127doG7HCSnk1N5obyUhh17S0J6Uj2qra0VfQS5NDk5WfxsdaQ5RSm5/Nq1a6BcooCgDdoOa8LaCmlO4dnb2yuSjR9/4udJ78TwlBrUpHyp93MIRIxAAK6uruI5yfIEAbXNVCANBoNoYmi97XAIRHTU1dWFhoYGAWa76Etz6pDIZVSRCYRaM9vhEIiYgu6GGIF6738adHIqkpQW5PaJQHbMMNEY+ZvcR43hl/xuq0/1rKKiQgQBFc7ExETbzyanQLaa/3Fu+gvPtDlHcIgPYQAAAABJRU5ErkJggg=="

var menuBarIcons = mustBuildIcons(menuBarIconBase64)

type stateIcons struct {
	active   []byte
	inactive []byte
}

func iconPNG(active bool) []byte {
	if active {
		return menuBarIcons.active
	}
	return menuBarIcons.inactive
}

func mustBuildIcons(encoded string) stateIcons {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		panic("invalid embedded menu bar icon: " + err.Error())
	}
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		panic("invalid embedded menu bar PNG: " + err.Error())
	}
	background := findExteriorBackground(source)
	return stateIcons{
		active:   mustEncodeSilhouette(source, background, 255),
		inactive: mustEncodeSilhouette(source, background, 105),
	}
}

// findExteriorBackground flood-fills near-white pixels from the image edges.
// This removes the official PNG's opaque white canvas while retaining white
// areas enclosed by the llama outline as part of the solid silhouette.
func findExteriorBackground(source image.Image) []bool {
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	background := make([]bool, width*height)
	queue := make([]image.Point, 0, 2*(width+height))

	add := func(x, y int) {
		index := (y-bounds.Min.Y)*width + x - bounds.Min.X
		if background[index] || !isBackgroundPixel(source.At(x, y)) {
			return
		}
		background[index] = true
		queue = append(queue, image.Pt(x, y))
	}
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		add(x, bounds.Min.Y)
		add(x, bounds.Max.Y-1)
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		add(bounds.Min.X, y)
		add(bounds.Max.X-1, y)
	}

	directions := [...]image.Point{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}}
	for len(queue) > 0 {
		point := queue[0]
		queue = queue[1:]
		for _, direction := range directions {
			next := point.Add(direction)
			if next.In(bounds) {
				add(next.X, next.Y)
			}
		}
	}
	return background
}

func isBackgroundPixel(value color.Color) bool {
	pixel := color.NRGBAModel.Convert(value).(color.NRGBA)
	return pixel.A < 16 || (pixel.R >= 245 && pixel.G >= 245 && pixel.B >= 245)
}

func mustEncodeSilhouette(source image.Image, background []bool, opacity uint8) []byte {
	bounds := source.Bounds()
	icon := image.NewNRGBA(source.Bounds())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			index := (y-bounds.Min.Y)*bounds.Dx() + x - bounds.Min.X
			if background[index] {
				continue
			}
			_, _, _, alpha := source.At(x, y).RGBA()
			scaledAlpha := uint8((alpha * uint32(opacity)) / 0xffff)
			icon.SetNRGBA(x, y, color.NRGBA{A: scaledAlpha})
		}
	}

	var output bytes.Buffer
	if err := png.Encode(&output, icon); err != nil {
		panic("menu bar silhouette could not be encoded: " + err.Error())
	}
	return output.Bytes()
}
