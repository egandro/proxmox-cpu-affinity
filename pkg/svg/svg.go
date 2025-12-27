package svg

import (
	"bytes"
	_ "embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
)

const (
	cellSize           = 40
	paddingTop         = 100
	basePaddingLeft    = 80
	affinityTableGap   = 40
	affinityTableWidth = 300
	affinityRowHeight  = 40
)

func New(rankings []cpuinfo.CoreRanking, summary cpuinfo.TopologyStats, affinity map[int][]int, cpuName string, mode Mode) *Heatmap {
	h := &Heatmap{
		rankings:    rankings,
		affinity:    affinity,
		cpuName:     cpuName,
		mode:        mode,
		cpuToSocket: make(map[int]int),
		latencyMap:  make(map[int]map[int]float64),
	}
	h.stats = summary

	h.prepareData()
	h.dims = h.calculateDimensions()
	return h
}

func (h *Heatmap) prepareData() {
	for _, r := range h.rankings {
		h.cpuIDs = append(h.cpuIDs, r.CPU)
		if h.latencyMap[r.CPU] == nil {
			h.latencyMap[r.CPU] = make(map[int]float64)
		}
		h.latencyMap[r.CPU][r.CPU] = 0.0

		for _, n := range r.Ranking {
			h.cpuToSocket[n.CPU] = n.Socket
			h.latencyMap[r.CPU][n.CPU] = n.LatencyNS
		}
	}

	sort.Slice(h.cpuIDs, func(i, j int) bool {
		id1 := h.cpuIDs[i]
		id2 := h.cpuIDs[j]
		s1 := h.cpuToSocket[id1]
		s2 := h.cpuToSocket[id2]
		if s1 != s2 {
			return s1 < s2
		}
		return id1 < id2
	})
}

func (h *Heatmap) calculateDimensions() heatmapDimensions {
	var dims heatmapDimensions
	n := len(h.cpuIDs)

	// Estimate text width requirements to avoid cropping long CPU names or stats
	// Title: font-size 20, bold (~12px/char)
	titleW := len(h.cpuName) * 12
	// Stats: font-size 14 (~8px/char)
	statsW := len(h.statisticsText()) * 8

	reqTextW := titleW
	if statsW > reqTextW {
		reqTextW = statsW
	}
	reqTextW += 40

	dims.paddingLeft = basePaddingLeft
	heatmapW := n * cellSize
	reqHeatmapW := dims.paddingLeft + heatmapW + 20

	dims.width = reqHeatmapW
	if reqTextW > dims.width {
		dims.width = reqTextW
		dims.paddingLeft += (dims.width - reqHeatmapW) / 2
	}

	dims.height = paddingTop + n*cellSize + 20

	if h.mode == ModeAffinity {
		dims.width += affinityTableGap + affinityTableWidth

		affinityHeaderHeight := 60
		affinityContentHeight := len(h.affinity) * affinityRowHeight
		if len(h.affinity) == 0 {
			affinityContentHeight = affinityRowHeight
		}
		totalAffinityHeight := affinityHeaderHeight + affinityContentHeight
		reqHeight := (paddingTop - 10) + totalAffinityHeight + 20
		if reqHeight > dims.height {
			dims.height = reqHeight
		}
	}

	return dims
}

func (h *Heatmap) specsText() string {
	numSockets := h.stats.SocketCount
	numCPUs := len(h.cpuIDs)
	socketStr := "Socket"
	if numSockets != 1 {
		socketStr = "Sockets"
	}

	cpuStr := "CPU"
	if numCPUs != 1 {
		cpuStr = "CPUs"
	}
	return fmt.Sprintf("%d %s | %d %s", numSockets, socketStr, numCPUs, cpuStr)
}

func (h *Heatmap) statisticsText() string {
	return fmt.Sprintf("Min: %.2f ns | Max: %.2f ns | Median: %.2f ns | Mean: %.2f ns", h.stats.MinLatencyNS, h.stats.MaxLatencyNS, h.stats.MedianLatencyNS, h.stats.MeanLatencyNS)
}

func (h *Heatmap) Generate() (string, error) {
	if len(h.rankings) == 0 {
		return "", fmt.Errorf("no ranking data available")
	}

	data := svgData{
		Width:   h.dims.width,
		Height:  h.dims.height,
		CenterX: h.dims.width / 2,
		Title:   h.cpuName,
		Specs:   h.specsText(),
		Stats:   h.statisticsText(),
	}

	for row, src := range h.cpuIDs {
		y := paddingTop + row*cellSize
		data.RowLabels = append(data.RowLabels, svgLabel{
			X:    h.dims.paddingLeft - 10,
			Y:    y + cellSize/2,
			Text: fmt.Sprintf("CPU %d", src),
		})

		if row > 0 && h.cpuToSocket[src] != h.cpuToSocket[h.cpuIDs[row-1]] {
			lineY := paddingTop + row*cellSize
			// Calculate heatmap width to ensure line stops at the edge of the heatmap
			heatmapWidth := len(h.cpuIDs) * cellSize
			lineX2 := h.dims.paddingLeft + heatmapWidth
			data.HLines = append(data.HLines, svgLine{
				X1: h.dims.paddingLeft,
				Y1: lineY,
				X2: lineX2,
				Y2: lineY,
			})
		}

		for col, dst := range h.cpuIDs {
			x := h.dims.paddingLeft + col*cellSize

			if row == 0 {
				data.ColLabels = append(data.ColLabels, svgLabel{
					X:    x + cellSize/2,
					Y:    paddingTop - 10,
					Text: fmt.Sprintf("%d", dst),
				})
			}

			if row == 0 && col > 0 && h.cpuToSocket[dst] != h.cpuToSocket[h.cpuIDs[col-1]] {
				lineX := h.dims.paddingLeft + col*cellSize
				data.VLines = append(data.VLines, svgLine{
					X1: lineX,
					Y1: paddingTop,
					X2: lineX,
					Y2: h.dims.height - 20,
				})
			}

			lat := h.latencyMap[src][dst]
			fill := "#eeeeee"
			textColor := "black"

			if src != dst {
				ratio := 0.0
				if h.stats.MaxLatencyNS > 0 {
					ratio = lat / h.stats.MaxLatencyNS
				}
				c, tc := calculateCellColor(ratio)
				fill = c.String()
				textColor = tc
			}

			data.Cells = append(data.Cells, svgCell{
				X:         x,
				Y:         y,
				Width:     cellSize,
				Height:    cellSize,
				Fill:      fill,
				TextColor: textColor,
				Text:      fmt.Sprintf("%.0f", lat),
				TextX:     x + cellSize/2,
				TextY:     y + cellSize/2,
			})
		}
	}

	if h.mode == ModeAffinity {
		data.ShowAffinity = true
		data.AffinityX = h.dims.paddingLeft + len(h.cpuIDs)*cellSize + affinityTableGap
		data.AffinityY = paddingTop + 20

		var vmids []int
		for vmid := range h.affinity {
			vmids = append(vmids, vmid)
		}
		sort.Ints(vmids)

		y := 30
		for i, vmid := range vmids {
			cpus := h.affinity[vmid]
			var cpuStrs []string
			for _, cpu := range cpus {
				cpuStrs = append(cpuStrs, strconv.Itoa(cpu))
			}

			fill := "white"
			if i%2 == 1 {
				fill = "#f9f9f9"
			}

			data.AffinityRows = append(data.AffinityRows, affinityRow{
				Y: y, VMID: strconv.Itoa(vmid), CPUs: strings.Join(cpuStrs, ","), LineY: y + 20,
				RectY: y - 20, Fill: fill,
			})
			y += affinityRowHeight
		}
	}

	tmpl, err := template.New("svg").Parse(svgTemplateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse SVG template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute SVG template: %w", err)
	}

	return buf.String(), nil
}

//go:embed templates/heatmap.svg.tmpl
var svgTemplateStr string
