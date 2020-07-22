package main

import (
	"flag"
	"fmt"
	"os"

	mp "github.com/livesense-inc/mackerel-plugin-buddyinfo/internal"
)

func main() {
	optVersion := flag.Bool("version", false, "Show plugin version")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	optPathBuddyinfo := flag.String("buddyinfo", mp.PathBuddyinfo, "buddyinfo file path")
	optPrefix := flag.String("metric-key-prefix", mp.MackerelKeyPrefix, "Metric key prefix")
	optVerbose := flag.Bool("verbose", false, "Verbose output")
	flag.Parse()

	if *optVersion {
		s := fmt.Sprintf("%s (rev:%s)", version, gitcommit)
		fmt.Printf("mackerel-plugin-buddyinfo: %s\n", s)
		os.Exit(0)
	}

	var plugin mp.BuddyinfoPlugin
	plugin.Prefix = *optPrefix
	switch {
	case *optVerbose:
		plugin.RunMode = mp.VerboseMode
	default:
		plugin.RunMode = mp.CompactMode
	}

	if *optPathBuddyinfo != "" {
		plugin.PathBuddyinfo = *optPathBuddyinfo
	} else {
		plugin.PathBuddyinfo = mp.PathBuddyinfo
	}

	if *optTempfile != "" {
		plugin.Tempfile = *optTempfile
	} else {
		plugin.Tempfile = ""
	}

	plugin.Do()
}
