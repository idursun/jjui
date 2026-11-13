package theme

import (
	"math"
)

// Minimum is the contrast target for ordinary text.
const Minimum = 4.5

// Luminance returns relative luminance in linear light.
func (c RGB) Luminance() float64 {
	var linear [3]float64
	for i, v := range c {
		s := float64(v) / 255
		if s <= 0.04045 {
			linear[i] = s / 12.92
		} else {
			linear[i] = math.Pow((s+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*linear[0] + 0.7152*linear[1] + 0.0722*linear[2]
}

// Ratio returns the contrast ratio of two resolved colours.
func Ratio(a, b RGB) float64 {
	l1, l2 := a.Luminance(), b.Luminance()
	return (max(l1, l2) + 0.05) / (min(l1, l2) + 0.05)
}

// Toward finds the smallest tint or shade that reaches the target.
// The endpoint must satisfy the ratio. Search the rounded, displayed colours,
// not just floating-point values which may round below the requested contrast.
func Toward(original, target, against RGB, ratio float64) RGB {
	low, high := 0.0, 1.0
	for range 16 {
		mid := (low + high) / 2
		if Ratio(original.mix(target, mid), against) >= ratio {
			high = mid
		} else {
			low = mid
		}
	}
	return original.mix(target, high)
}

// Foreground returns the nearest tint or shade meeting Minimum contrast.
// Readable colours are returned unchanged.
func Foreground(foreground, background RGB) RGB {
	if Ratio(foreground, background) >= Minimum {
		return foreground
	}
	var best RGB
	bestDistance := math.Inf(1)
	for _, edge := range []RGB{{0, 0, 0}, {255, 255, 255}} {
		if Ratio(edge, background) < Minimum {
			continue
		}
		candidate := Toward(foreground, edge, background, Minimum)
		distance := 0.0
		for i := range candidate {
			delta := float64(candidate[i]) - float64(foreground[i])
			distance += delta * delta
		}
		if distance < bestDistance {
			best, bestDistance = candidate, distance
		}
	}
	return best
}

// HighlightTextContrast leaves headroom for syntax colours.
const HighlightTextContrast = 7.0

func HighlightBackground(background, surface RGB) RGB {
	black, white := RGB{0, 0, 0}, RGB{255, 255, 255}
	edge := white
	if Ratio(black, surface) > Ratio(white, surface) {
		edge = black
	}
	if Ratio(edge, background) >= HighlightTextContrast ||
		Ratio(edge, surface) < HighlightTextContrast {
		return background
	}
	return Toward(background, surface, edge, HighlightTextContrast)
}
