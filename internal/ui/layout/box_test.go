package layout

import (
	"testing"

	"github.com/charmbracelet/x/cellbuf"
)

func TestNewBox(t *testing.T) {
	r := cellbuf.Rect(0, 0, 100, 50)
	b := NewBox(r)
	if b.R != r {
		t.Errorf("NewBox() = %v, want %v", b.R, r)
	}
}

func TestFixedSpec(t *testing.T) {
	tests := []struct {
		name  string
		fixed int
		total int
		want  int
	}{
		{"normal", 10, 100, 10},
		{"zero", 0, 100, 0},
		{"negative", -5, 100, 0},
		{"overflow", 150, 100, 100},
		{"exact", 100, 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := Fixed(tt.fixed)
			got := spec.calc(tt.total, 0, 0)
			if got != tt.want {
				t.Errorf("Fixed(%d).calc(%d) = %d, want %d", tt.fixed, tt.total, got, tt.want)
			}
		})
	}
}

func TestPctSpec(t *testing.T) {
	tests := []struct {
		name  string
		pct   int
		total int
		want  int
	}{
		{"zero", 0, 100, 0},
		{"quarter", 25, 100, 25},
		{"half", 50, 100, 50},
		{"full", 100, 100, 100},
		{"negative", -10, 100, 0},
		{"overflow", 150, 100, 100},
		{"rounding", 33, 100, 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := Percent(tt.pct)
			got := spec.calc(tt.total, 0, 0)
			if got != tt.want {
				t.Errorf("Percent(%d).calc(%d) = %d, want %d", tt.pct, tt.total, got, tt.want)
			}
		})
	}
}

func TestFillSpec(t *testing.T) {
	tests := []struct {
		name       string
		weight     float64
		total      int
		remaining  int
		fillWeight float64
		want       int
	}{
		{"single_fill", 1.0, 100, 50, 1.0, 50},
		{"half_weight", 1.0, 100, 50, 2.0, 25},
		{"double_weight", 2.0, 100, 50, 3.0, 33},
		{"no_remaining", 1.0, 100, 0, 1.0, 0},
		{"zero_weight", 0.0, 100, 50, 1.0, 0},
		{"negative_weight", -1.0, 100, 50, 1.0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := Fill(tt.weight)
			got := spec.calc(tt.total, tt.remaining, tt.fillWeight)
			if got != tt.want {
				t.Errorf("Fill(%f).calc(%d, %d, %f) = %d, want %d",
					tt.weight, tt.total, tt.remaining, tt.fillWeight, got, tt.want)
			}
		})
	}
}

func TestBoxInset(t *testing.T) {
	b := NewBox(cellbuf.Rect(10, 10, 50, 30))
	inset := b.Inset(2)
	want := cellbuf.Rect(12, 12, 46, 26)

	if inset.R != want {
		t.Errorf("Box.Inset(2) = %v, want %v", inset.R, want)
	}
}

func TestBoxV_SingleFixed(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 100))
	boxes := b.V(Fixed(20))

	if len(boxes) != 1 {
		t.Fatalf("V() returned %d boxes, want 1", len(boxes))
	}

	want := cellbuf.Rect(0, 0, 100, 20)
	if boxes[0].R != want {
		t.Errorf("V(Fixed(20)) = %v, want %v", boxes[0].R, want)
	}
}

func TestBoxV_TwoFixed(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 100))
	boxes := b.V(Fixed(20), Fixed(30))

	if len(boxes) != 2 {
		t.Fatalf("V() returned %d boxes, want 2", len(boxes))
	}

	want1 := cellbuf.Rect(0, 0, 100, 20)
	want2 := cellbuf.Rect(0, 20, 100, 30)

	if boxes[0].R != want1 {
		t.Errorf("boxes[0] = %v, want %v", boxes[0].R, want1)
	}
	if boxes[1].R != want2 {
		t.Errorf("boxes[1] = %v, want %v", boxes[1].R, want2)
	}
}

func TestBoxV_FixedAndFill(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 100))
	boxes := b.V(Fixed(10), Fill(1), Fixed(10))

	if len(boxes) != 3 {
		t.Fatalf("V() returned %d boxes, want 3", len(boxes))
	}

	if boxes[0].R.Dy() != 10 {
		t.Errorf("boxes[0] height = %d, want 10", boxes[0].R.Dy())
	}
	if boxes[1].R.Dy() != 80 {
		t.Errorf("boxes[1] height = %d, want 80", boxes[1].R.Dy())
	}
	if boxes[2].R.Dy() != 10 {
		t.Errorf("boxes[2] height = %d, want 10", boxes[2].R.Dy())
	}
}

func TestBoxV_MultipleFillWeights(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 100))
	boxes := b.V(Fixed(10), Fill(1), Fill(2))

	if len(boxes) != 3 {
		t.Fatalf("V() returned %d boxes, want 3", len(boxes))
	}

	if boxes[0].R.Dy() != 10 {
		t.Errorf("boxes[0] height = %d, want 10", boxes[0].R.Dy())
	}
	if boxes[1].R.Dy() != 30 {
		t.Errorf("boxes[1] height = %d, want 30", boxes[1].R.Dy())
	}
	if boxes[2].R.Dy() != 60 {
		t.Errorf("boxes[2] height = %d, want 60", boxes[2].R.Dy())
	}
}

func TestBoxV_Percentage(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 100))
	boxes := b.V(Percent(20), Percent(30))

	if len(boxes) != 2 {
		t.Fatalf("V() returned %d boxes, want 2", len(boxes))
	}

	if boxes[0].R.Dy() != 20 {
		t.Errorf("boxes[0] height = %d, want 20", boxes[0].R.Dy())
	}
	if boxes[1].R.Dy() != 30 {
		t.Errorf("boxes[1] height = %d, want 30", boxes[1].R.Dy())
	}
}

func TestBoxV_MixedSpecs(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 80, 24))
	boxes := b.V(Fixed(1), Fill(1), Fixed(2))

	if len(boxes) != 3 {
		t.Fatalf("V() returned %d boxes, want 3", len(boxes))
	}

	if boxes[0].R.Dy() != 1 {
		t.Errorf("header height = %d, want 1", boxes[0].R.Dy())
	}
	if boxes[1].R.Dy() != 21 {
		t.Errorf("body height = %d, want 21", boxes[1].R.Dy())
	}
	if boxes[2].R.Dy() != 2 {
		t.Errorf("footer height = %d, want 2", boxes[2].R.Dy())
	}

	if boxes[0].R.Min.Y != 0 || boxes[0].R.Max.Y != 1 {
		t.Errorf("header Y range = [%d, %d], want [0, 1]", boxes[0].R.Min.Y, boxes[0].R.Max.Y)
	}
	if boxes[1].R.Min.Y != 1 || boxes[1].R.Max.Y != 22 {
		t.Errorf("body Y range = [%d, %d], want [1, 22]", boxes[1].R.Min.Y, boxes[1].R.Max.Y)
	}
	if boxes[2].R.Min.Y != 22 || boxes[2].R.Max.Y != 24 {
		t.Errorf("footer Y range = [%d, %d], want [22, 24]", boxes[2].R.Min.Y, boxes[2].R.Max.Y)
	}
}

func TestBoxH_TwoFixed(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 50))
	boxes := b.H(Fixed(20), Fixed(30))

	if len(boxes) != 2 {
		t.Fatalf("H() returned %d boxes, want 2", len(boxes))
	}

	want1 := cellbuf.Rect(0, 0, 20, 50)
	want2 := cellbuf.Rect(20, 0, 30, 50)

	if boxes[0].R != want1 {
		t.Errorf("boxes[0] = %v, want %v", boxes[0].R, want1)
	}
	if boxes[1].R != want2 {
		t.Errorf("boxes[1] = %v, want %v", boxes[1].R, want2)
	}
}

func TestBoxH_SidebarMainSidebar(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 50))
	boxes := b.H(Fixed(20), Fill(1), Fixed(15))

	if len(boxes) != 3 {
		t.Fatalf("H() returned %d boxes, want 3", len(boxes))
	}

	if boxes[0].R.Dx() != 20 {
		t.Errorf("left width = %d, want 20", boxes[0].R.Dx())
	}
	if boxes[1].R.Dx() != 65 {
		t.Errorf("main width = %d, want 65", boxes[1].R.Dx())
	}
	if boxes[2].R.Dx() != 15 {
		t.Errorf("right width = %d, want 15", boxes[2].R.Dx())
	}
}

func TestBoxCutTop(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 100)) // (0,0)-(100,100)
	top, rest := b.CutTop(20)

	wantTop := cellbuf.Rect(0, 0, 100, 20)
	wantRest := cellbuf.Rect(0, 20, 100, 80)

	if top.R != wantTop {
		t.Errorf("top = %v, want %v", top.R, wantTop)
	}
	if rest.R != wantRest {
		t.Errorf("rest = %v, want %v", rest.R, wantRest)
	}
}

func TestBoxCutBottom(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 100)) // (0,0)-(100,100)
	rest, bottom := b.CutBottom(20)

	wantRest := cellbuf.Rect(0, 0, 100, 80)
	wantBottom := cellbuf.Rect(0, 80, 100, 20)

	if rest.R != wantRest {
		t.Errorf("rest = %v, want %v", rest.R, wantRest)
	}
	if bottom.R != wantBottom {
		t.Errorf("bottom = %v, want %v", bottom.R, wantBottom)
	}
}

func TestBoxCutLeft(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 50)) // (0,0)-(100,50)
	left, rest := b.CutLeft(25)

	wantLeft := cellbuf.Rect(0, 0, 25, 50)
	wantRest := cellbuf.Rect(25, 0, 75, 50)

	if left.R != wantLeft {
		t.Errorf("left = %v, want %v", left.R, wantLeft)
	}
	if rest.R != wantRest {
		t.Errorf("rest = %v, want %v", rest.R, wantRest)
	}
}

func TestBoxCutRight(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 50)) // (0,0)-(100,50)
	rest, right := b.CutRight(25)

	wantRest := cellbuf.Rect(0, 0, 75, 50)
	wantRight := cellbuf.Rect(75, 0, 25, 50)

	if rest.R != wantRest {
		t.Errorf("rest = %v, want %v", rest.R, wantRest)
	}
	if right.R != wantRight {
		t.Errorf("right = %v, want %v", right.R, wantRight)
	}
}

func TestBoxCenter(t *testing.T) {
	tests := []struct {
		name string
		box  Box
		w, h int
		want cellbuf.Rectangle
	}{
		{
			name: "centered in 100x100",
			box:  NewBox(cellbuf.Rect(0, 0, 100, 100)),
			w:    60,
			h:    40,
			want: cellbuf.Rect(20, 30, 60, 40),
		},
		{
			name: "centered in offset box",
			box:  NewBox(cellbuf.Rect(10, 10, 100, 100)),
			w:    60,
			h:    40,
			want: cellbuf.Rect(30, 40, 60, 40),
		},
		{
			name: "overflow width",
			box:  NewBox(cellbuf.Rect(0, 0, 50, 50)),
			w:    100,
			h:    20,
			want: cellbuf.Rect(0, 15, 50, 20),
		},
		{
			name: "overflow height",
			box:  NewBox(cellbuf.Rect(0, 0, 50, 50)),
			w:    20,
			h:    100,
			want: cellbuf.Rect(15, 0, 20, 50),
		},
		{
			name: "zero size",
			box:  NewBox(cellbuf.Rect(0, 0, 100, 100)),
			w:    0,
			h:    0,
			want: cellbuf.Rect(50, 50, 0, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.box.Center(tt.w, tt.h)
			if got.R != tt.want {
				t.Errorf("Center(%d, %d) = %v, want %v", tt.w, tt.h, got.R, tt.want)
			}
		})
	}
}

func TestBoxV_EmptySpecs(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 100))
	boxes := b.V()

	if len(boxes) != 1 {
		t.Fatalf("V() with no specs returned %d boxes, want 1", len(boxes))
	}

	if boxes[0].R != b.R {
		t.Errorf("V() with no specs = %v, want %v", boxes[0].R, b.R)
	}
}

func TestBoxV_ZeroHeight(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 0))
	boxes := b.V(Fixed(10), Fill(1))

	if len(boxes) != 2 {
		t.Fatalf("V() returned %d boxes, want 2", len(boxes))
	}

	for i, box := range boxes {
		if box.R.Dy() != 0 {
			t.Errorf("boxes[%d] height = %d, want 0", i, box.R.Dy())
		}
	}
}

func TestBoxH_ZeroWidth(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 0, 100))
	boxes := b.H(Fixed(10), Fill(1))

	if len(boxes) != 2 {
		t.Fatalf("H() returned %d boxes, want 2", len(boxes))
	}

	for i, box := range boxes {
		if box.R.Dx() != 0 {
			t.Errorf("boxes[%d] width = %d, want 0", i, box.R.Dx())
		}
	}
}

func TestBoxV_Overflow(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 100))
	boxes := b.V(Fixed(60), Fixed(70))

	if len(boxes) != 2 {
		t.Fatalf("V() returned %d boxes, want 2", len(boxes))
	}

	if boxes[0].R.Dy() != 60 {
		t.Errorf("boxes[0] height = %d, want 60", boxes[0].R.Dy())
	}
	if boxes[1].R.Dy() != 40 {
		t.Errorf("boxes[1] height = %d, want 40", boxes[1].R.Dy())
	}
}

func TestBoxV_RoundingRemainder(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 100))
	boxes := b.V(Fill(1), Fill(1), Fill(1))

	if len(boxes) != 3 {
		t.Fatalf("V() returned %d boxes, want 3", len(boxes))
	}

	total := 0
	for i, box := range boxes {
		h := box.R.Dy()
		t.Logf("boxes[%d] height = %d", i, h)
		total += h
	}

	if total != 100 {
		t.Errorf("total height = %d, want 100", total)
	}

	if boxes[2].R.Dy() < boxes[0].R.Dy() {
		t.Errorf("last box height %d < first box height %d", boxes[2].R.Dy(), boxes[0].R.Dy())
	}
}

func TestBoxChaining(t *testing.T) {
	b := NewBox(cellbuf.Rect(0, 0, 100, 100))

	boxes := b.Inset(5).V(Fixed(10), Fill(1), Fixed(10))
	middle := boxes[1]
	cols := middle.H(Fixed(20), Fill(1))

	if middle.R.Dy() != 70 {
		t.Errorf("middle height = %d, want 70", middle.R.Dy())
	}

	if cols[0].R.Dx() != 20 {
		t.Errorf("left column width = %d, want 20", cols[0].R.Dx())
	}
	if cols[1].R.Dx() != 70 {
		t.Errorf("right column width = %d, want 70", cols[1].R.Dx())
	}
}
