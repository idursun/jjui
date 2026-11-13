package theme

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContrastRatioReferenceValues(t *testing.T) {
	assert.Equal(t, 21.0, Ratio(RGB{}, RGB{255, 255, 255}))
	assert.Equal(t, 1.0, Ratio(RGB{127, 127, 127}, RGB{127, 127, 127}))
	// Relative luminance of sRGB red is 0.2126.
	assert.InDelta(t, 5.252, Ratio(RGB{255, 0, 0}, RGB{}), 0.00001)
}

func TestReadableForegroundAcrossColourPairs(t *testing.T) {
	random := rand.New(rand.NewPCG(1, 2))
	for range 10000 {
		fg := RGB{uint8(random.Uint32()), uint8(random.Uint32()), uint8(random.Uint32())}
		bg := RGB{uint8(random.Uint32()), uint8(random.Uint32()), uint8(random.Uint32())}
		got := Foreground(fg, bg)
		require.GreaterOrEqual(t, Ratio(got, bg), Minimum, "fg=%v bg=%v corrected=%v", fg, bg, got)
		if Ratio(fg, bg) >= Minimum {
			require.Equal(t, fg, got)
		}
	}
}

func TestScreenshotContrast(t *testing.T) {
	// Sampled solid foreground/background pixels, excluding antialiasing.
	foreground := RGB{182, 149, 243}
	surface := RGB{40, 42, 53}
	for _, tt := range []struct {
		name             string
		background       RGB
		originalContrast float64
	}{
		{"removed word", RGB{134, 63, 67}, 3.059},
		{"added word", RGB{79, 139, 87}, 1.662},
		{"removed line", RGB{83, 50, 58}, 4.553},
		{"added line", RGB{56, 88, 66}, 3.248},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.originalContrast, Ratio(foreground, tt.background), 0.001)
			bg := HighlightBackground(tt.background, surface)
			fg := Foreground(foreground, bg)
			assert.GreaterOrEqual(t, Ratio(fg, bg), Minimum)
			// Keep this purple recognisably purple and light on the dark surface.
			assert.GreaterOrEqual(t, fg[0], foreground[0])
			assert.Greater(t, fg[2], fg[0])
			assert.Greater(t, fg[0], fg[1])
			if tt.originalContrast >= Minimum {
				assert.Equal(t, foreground, fg)
				assert.Equal(t, tt.background, bg)
			}
		})
	}
}
