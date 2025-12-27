package svg

import "github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"

type Mode int

const (
	ModeDefault Mode = iota
	ModeAffinity
)

type Heatmap struct {
	rankings []cpuinfo.CoreRanking
	affinity map[int][]int
	cpuName  string
	mode     Mode

	cpuIDs      []int
	cpuToSocket map[int]int
	latencyMap  map[int]map[int]float64

	stats cpuinfo.TopologyStats
	dims  heatmapDimensions
}

type svgLabel struct {
	X, Y int
	Text string
}

type svgLine struct {
	X1, Y1, X2, Y2 int
}

type svgCell struct {
	X, Y, Width, Height int
	Fill, TextColor     string
	Text                string
	TextX, TextY        int
}

type svgData struct {
	Width, Height, CenterX int
	Title, Specs, Stats    string
	RowLabels, ColLabels   []svgLabel
	HLines, VLines         []svgLine
	Cells                  []svgCell

	ShowAffinity bool
	AffinityX    int
	AffinityY    int
	AffinityRows []affinityRow
}

type affinityRow struct {
	Y     int
	VMID  string
	CPUs  string
	LineY int
	RectY int
	Fill  string
}

type heatmapDimensions struct {
	width       int
	height      int
	paddingLeft int
}
