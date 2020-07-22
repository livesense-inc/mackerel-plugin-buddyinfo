package mpbuddyinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraphDefinition(t *testing.T) {
	var p BuddyinfoPlugin
	p.PathBuddyinfo = "../test/buddyinfo.sample"
	p.RunMode = CompactMode
	graphs := p.GraphDefinition()

	assert.EqualValues(t, "Buddyinfo Available Pages Summary (Percentage)", graphs["available_pages_summary_pct.#"].Label)
	assert.EqualValues(t, "Buddyinfo Average Size", graphs["average_size"].Label)
	assert.NotEqual(t, "Buddyinfo Available Pages", graphs["available_pages.#"].Label)

	p.RunMode = VerboseMode
	graphs = p.GraphDefinition()
	assert.EqualValues(t, "Buddyinfo Available Pages Summary (Percentage)", graphs["available_pages_summary_pct.#"].Label)
	assert.EqualValues(t, "Buddyinfo Average Size", graphs["average_size"].Label)
	assert.EqualValues(t, "Buddyinfo Available Pages", graphs["available_pages.#"].Label)

	assert.EqualValues(t, 3, len(graphs["available_pages_summary_pct.#"].Metrics))
	assert.EqualValues(t, 4, len(graphs["average_size"].Metrics))
	assert.EqualValues(t, 11, len(graphs["available_pages.#"].Metrics))
}

func TestFetchMetrics(t *testing.T) {

	var p BuddyinfoPlugin
	p.PathBuddyinfo = "../test/buddyinfo.sample"
	stat, err := p.FetchMetrics()
	if err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 58, stat["available_pages.Node0_Normal.256K"])
	assert.EqualValues(t, 686881.3913043478, stat["average_size.Node0_DMA"])
	assert.EqualValues(t, 9.310489363931678, stat["available_pages_summary_pct.Node0_DMA32.middle"])
}
