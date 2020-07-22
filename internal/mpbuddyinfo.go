package mpbuddyinfo

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	mp "github.com/mackerelio/go-mackerel-plugin-helper"
	"github.com/mackerelio/golib/logging"
)

var logger = logging.GetLogger("metrics.plugin.buddyinfo")

// MackerelKeyPrefix is default value of Mackerel Key Prefix string
var MackerelKeyPrefix = "buddyinfo"

// PathBuddyinfo is path to buddyinfo file.
var PathBuddyinfo = "/proc/buddyinfo"

// BuddySizes is mapping of culumn names and page sizes
var BuddySizes = map[string]uint64{
	"4K":   4 * 1024,
	"8K":   8 * 1024,
	"16K":  16 * 1024,
	"32K":  32 * 1024,
	"64K":  64 * 1024,
	"128K": 128 * 1024,
	"256K": 256 * 1024,
	"512K": 512 * 1024,
	"1M":   1 * 1024 * 1024,
	"2M":   2 * 1024 * 1024,
	"4M":   4 * 1024 * 1024,
}

// BuddySizeCategories is category for summarizing
var BuddySizeCategories = map[string]string{
	"4K":   "small",
	"8K":   "small",
	"16K":  "small",
	"32K":  "small",
	"64K":  "middle",
	"128K": "middle",
	"256K": "middle",
	"512K": "middle",
	"1M":   "large",
	"2M":   "large",
	"4M":   "large",
}

var (
	// CompactMode is default mode
	CompactMode = 0
	// VerboseMode is verbose output mode
	VerboseMode = 5
)

func readFile(file string) (lines []string, err error) {
	var fp *os.File

	fp, err = os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, nil
}

/*
 * regex to parse buddyinfo
 * example:
 * Node 0, zone      DMA      3      1      3      2      1      3      2      2      3      1      2
 */
var com = regexp.MustCompile(`(?m)(\s+)|(\#.*$)`)
var r = com.ReplaceAllString(
	`^
  (?P<node>[^,]+),
  \s*zone\s+(?P<zone>[^\s]+)
  \s+(?P<4K>\d+)
  \s+(?P<8K>\d+)
	\s+(?P<16K>\d+)
	\s+(?P<32K>\d+)
	\s+(?P<64K>\d+)
	\s+(?P<128K>\d+)
	\s+(?P<256K>\d+)
	\s+(?P<512K>\d+)
	\s+(?P<1M>\d+)
	\s+(?P<2M>\d+)
	\s+(?P<4M>\d+)
	.*                # do not treat section after 4M
	$`, "")
var reg = regexp.MustCompile(r)

// BuddyinfoPlugin mackerel plugin for buddyinfo
type BuddyinfoPlugin struct {
	Tempfile      string
	Prefix        string
	RunMode       int
	PathBuddyinfo string
}

func (p BuddyinfoPlugin) fetchBuddyinfo() (result map[string]interface{}, err error) {
	result = make(map[string]interface{})

	lines, err := readFile(p.PathBuddyinfo)
	if err != nil {
		logger.Warningf("Failed to read buddyinfo: %s", err.Error())
		return
	}

	for _, line := range lines {
		line = strings.Trim(line, " \n")
		m := reg.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}
		parsed := make(map[string]string)
		for i, name := range reg.SubexpNames() {
			if i != 0 && name != "" {
				parsed[name] = m[i]
			}
		}

		if _, ok := parsed["node"]; !ok {
			logger.Infof("Failed to parse: %s", line)
			continue
		}
		if _, ok := parsed["zone"]; !ok {
			logger.Infof("Failed to parse: %s", line)
			continue
		}

		zoneName := fmt.Sprintf("%s_%s",
			strings.ReplaceAll(parsed["node"], " ", ""),
			strings.ReplaceAll(parsed["zone"], " ", ""))
		delete(parsed, "node")
		delete(parsed, "zone")

		sumAvailable := uint64(0)
		sumPages := uint64(0)
		summary := make(map[string]uint64)

		for k, v := range parsed {
			pageCnt, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				logger.Infof("Failed to parse: %s", v)
				continue
			}

			// insert available_pages metrics
			key := fmt.Sprintf("available_pages.%s.%s", zoneName, k)
			result[key] = pageCnt

			// // insert available_pages_summary_pct metrics
			// key = fmt.Sprintf("available_pages_summary_pct.%s.%s", zoneName, BuddySizeCategories[k])
			key = BuddySizeCategories[k]
			if _, ok := summary[key]; !ok {
				summary[key] = uint64(0)
			}
			summary[key] = summary[key] + pageCnt

			// calculate average_size statitics
			sumAvailable += BuddySizes[k] * pageCnt
			sumPages += pageCnt
		}

		// insert average_size metrics
		key := fmt.Sprintf("average_size.%s", zoneName)
		result[key] = float64(sumAvailable) / float64(sumPages)

		// insert available_pages_summary_pct metrics
		for k, v := range summary {
			key = fmt.Sprintf("available_pages_summary_pct.%s.%s", zoneName, k)
			result[key] = float64(v) / float64(sumPages) * 100
		}
	}

	return result, nil
}

// MetricKeyPrefix interface for PluginWithPrefix
func (p BuddyinfoPlugin) MetricKeyPrefix() string {
	return p.Prefix
}

// FetchMetrics interface for PluginWithPrefix
func (p BuddyinfoPlugin) FetchMetrics() (map[string]interface{}, error) {
	return p.fetchBuddyinfo()
}

// GraphDefinition interface for PluginWithPrefix
func (p BuddyinfoPlugin) GraphDefinition() map[string]mp.Graphs {
	metricsAvailablePagesSummary := []mp.Metrics{}
	categories := make(map[string]struct{})
	for _, v := range BuddySizeCategories {
		categories[v] = struct{}{}
	}
	for k := range categories {
		metric := mp.Metrics{
			Name:         k,
			Label:        k,
			Type:         "float64",
			Diff:         false,
			Stacked:      true,
			AbsoluteName: true,
		}
		metricsAvailablePagesSummary = append(metricsAvailablePagesSummary, metric)
	}

	// extract zone names from result of fetching buddyinfo
	result, err := p.fetchBuddyinfo()
	if err != nil {
		logger.Errorf("cannot fetch buddyinfo: '%s'\n", err.Error())
		os.Exit(1)
	}
	zoneNames := []string{}
	for k := range result {
		if strings.HasPrefix(k, "average_size.") {
			zoneNames = append(zoneNames, k[len("average_size."):])
		}
	}

	metricsAverageSize := []mp.Metrics{}
	for _, k := range zoneNames {
		metric := mp.Metrics{
			Name:         k,
			Label:        k,
			Type:         "float64",
			Diff:         false,
			Stacked:      false,
			AbsoluteName: true,
		}
		metricsAverageSize = append(metricsAverageSize, metric)
	}

	var graphdef = map[string]mp.Graphs{
		"available_pages_summary_pct.#": {
			Label:   ("Buddyinfo Available Pages Summary (Percentage)"),
			Unit:    "percentage",
			Metrics: metricsAvailablePagesSummary,
		},
		"average_size": {
			Label:   ("Buddyinfo Average Size"),
			Unit:    "bytes",
			Metrics: metricsAverageSize,
		},
	}

	if p.RunMode >= VerboseMode {
		metricsAvailablePages := []mp.Metrics{}
		for k := range BuddySizes {
			metric := mp.Metrics{
				Name:         k,
				Label:        k,
				Type:         "uint64",
				Diff:         false,
				Stacked:      true,
				AbsoluteName: true,
			}
			metricsAvailablePages = append(metricsAvailablePages, metric)
		}

		graphdef["available_pages.#"] = mp.Graphs{
			Label:   ("Buddyinfo Available Pages"),
			Unit:    "integer",
			Metrics: metricsAvailablePages,
		}
	}

	return graphdef
}

// Do the plugin
func (p BuddyinfoPlugin) Do() {
	helper := mp.NewMackerelPlugin(p)
	if p.Tempfile != "" {
		helper.Tempfile = p.Tempfile
	} else {
		helper.SetTempfileByBasename("mackerel-plugin-buddyinfo")
	}

	helper.Run()
}
