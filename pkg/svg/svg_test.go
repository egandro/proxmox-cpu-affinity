package svg

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	t.Skip("Skipping dummy svg generation.")

	testDataDir := "testdata"
	resultDir := "testresult"

	// Ensure result directory exists
	err := os.MkdirAll(resultDir, 0755)
	require.NoError(t, err)

	files, err := filepath.Glob(filepath.Join(testDataDir, "*.json"))
	require.NoError(t, err)

	modes := []struct {
		mode     Mode
		modeName string
	}{
		{ModeDefault, "default"},
		{ModeAffinity, "affinity"},
	}

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(file)
			require.NoError(t, err)

			var rankings []cpuinfo.CoreRanking
			err = json.Unmarshal(content, &rankings)
			require.NoError(t, err)

			affinity := generateDummyAffinity(rankings)
			summary := cpuinfo.SummarizeRankings(rankings)

			for _, m := range modes {
				t.Run(m.modeName, func(t *testing.T) {
					heatmap := New(rankings, summary, affinity, "Test CPU Model: "+name, m.mode)
					svgContent, err := heatmap.Generate()
					assert.NoError(t, err)
					assert.NotEmpty(t, svgContent)

					baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
					outputFile := filepath.Join(resultDir, baseName+"-"+m.modeName+".svg")
					err = os.WriteFile(outputFile, []byte(svgContent), 0644)
					assert.NoError(t, err)
				})
			}
		})
	}
}

func generateDummyAffinity(rankings []cpuinfo.CoreRanking) map[int][]int {
	// Generate affinity data based on CPU count
	// Rule: Max 50% of total CPUs assigned
	// Rule: Max 1/3 of total CPUs per VM
	affinity := make(map[int][]int)
	var cpuIDs []int
	for _, r := range rankings {
		cpuIDs = append(cpuIDs, r.CPU)
	}
	sort.Ints(cpuIDs)

	numCores := len(cpuIDs)
	// Rule: Use at least 2/3 of total cores across all VMs
	targetUsedCores := (numCores * 2) / 3
	if targetUsedCores == 0 {
		targetUsedCores = 1
	}

	// Rule: Max 8 cores per VM
	maxPerVM := 8
	if numCores < maxPerVM {
		maxPerVM = numCores
	}

	used := 0
	vmid := 200

	// Create VMs until we have "used" enough cores (conceptually)
	for used < targetUsedCores {
		// Vary core count slightly for realism, but keep it simple for dummy data
		count := (vmid % 4) + 1 // 1 to 4 cores
		if count > maxPerVM {
			count = maxPerVM
		}

		var cpus []int
		for i := 0; i < count; i++ {
			// Pick cores in a way that spreads them out but allows some overlap/pattern
			// This simple logic just picks cores based on current usage index wrapping around
			coreIdx := (used + i) % numCores
			cpus = append(cpus, cpuIDs[coreIdx])
		}
		affinity[vmid] = cpus
		used += 1 // Increment used "slots" (not necessarily unique cores)
		vmid++
	}
	return affinity
}

func TestCreateDummyJsonTestData(t *testing.T) {
	t.Skip("Skipping create dummy json test.")
	// data from here https://github.com/nviennot/core-to-core-latency/tree/main/results

	sources := []struct {
		url     string
		sockets int
	}{
		{
			url:     "https://raw.githubusercontent.com/nviennot/core-to-core-latency/main/results/Loongson%203A5000HV%2C%202.5GHz%2C%204%20Cores%2C%202021-Q3.csv",
			sockets: 1,
		},
		{
			url:     "https://raw.githubusercontent.com/nviennot/core-to-core-latency/refs/heads/main/results/AMD%20Ryzen%209%207950X.csv",
			sockets: 1,
		},
		{
			url:     "https://raw.githubusercontent.com/nviennot/core-to-core-latency/refs/heads/main/results/Dual%20Intel(R)%20Xeon%20Gold%206242%20%40%202.8GHz.csv",
			sockets: 2,
		},
	}

	// the data is fake!

	testDataDir := "testdata"
	err := os.MkdirAll(testDataDir, 0755)
	require.NoError(t, err)

	for _, src := range sources {
		u, err := url.Parse(src.url)
		require.NoError(t, err)

		filename, err := url.QueryUnescape(filepath.Base(u.Path))
		require.NoError(t, err)

		t.Logf("Downloading %s...", filename)

		resp, err := http.Get(src.url)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Logf("Failed to download %s: %s", src.url, resp.Status)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		t.Logf("Downloaded %d bytes", len(body))

		r := csv.NewReader(strings.NewReader(string(body)))
		r.FieldsPerRecord = -1
		records, err := r.ReadAll()
		require.NoError(t, err)

		latencyMap := make(map[int]map[int]float64)
		for i, row := range records {
			for j, col := range row {
				if val, err := strconv.ParseFloat(col, 64); err == nil {
					if latencyMap[i] == nil {
						latencyMap[i] = make(map[int]float64)
					}
					latencyMap[i][j] = val
					if latencyMap[j] == nil {
						latencyMap[j] = make(map[int]float64)
					}
					latencyMap[j][i] = val
				}
			}
		}

		numCores := len(records)
		coresPerSocket := numCores
		if src.sockets > 0 {
			coresPerSocket = numCores / src.sockets
		}
		if coresPerSocket == 0 {
			coresPerSocket = 1
		}

		var rankings []cpuinfo.CoreRanking
		for i := 0; i < numCores; i++ {
			var neighbors []cpuinfo.Neighbor
			if lats, ok := latencyMap[i]; ok {
				for target, lat := range lats {
					socketID := 0
					if src.sockets > 1 {
						socketID = target / coresPerSocket
					}

					neighbors = append(neighbors, cpuinfo.Neighbor{
						CPU:       target,
						Socket:    socketID,
						Core:      target,
						LatencyNS: lat,
					})
				}
			}
			sort.Slice(neighbors, func(a, b int) bool {
				return neighbors[a].LatencyNS < neighbors[b].LatencyNS
			})
			rankings = append(rankings, cpuinfo.CoreRanking{
				CPU:     i,
				Ranking: neighbors,
			})
		}

		jsonData, err := json.MarshalIndent(rankings, "", "  ")
		require.NoError(t, err)

		filename = "fake-" + filename
		jsonFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".json"
		outputFile := filepath.Join(testDataDir, jsonFilename)
		err = os.WriteFile(outputFile, jsonData, 0644)
		require.NoError(t, err)
		t.Logf("Saved %s", outputFile)
	}
}
