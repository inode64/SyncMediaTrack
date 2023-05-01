package cmd

import (
	syncmediatrack "github.com/inode64/SyncMediaTrack/lib"
	"github.com/spf13/cobra"
)

var (
	dryRun     bool
	force      bool
	geoservice bool
	track      string
)

var rootCmd = &cobra.Command{
	Use:     "SyncMediaTack",
	Short:   "Synchronize Media Data from track GPX",
	Long:    `Using a gpx track, analyze a directory with images or movies and add the GPS positions`,
	Args:    cobra.MinimumNArgs(1),
	Version: "1.3",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Performs the actions without writing to the files")
	rootCmd.PersistentFlags().BoolVar(&force, "force", false, "Force update even overwriting previous GPS data")
	rootCmd.PersistentFlags().BoolVar(&geoservice, "geoservice", false, "Show location from GPS position from geocoding service of openstreetmap")
	rootCmd.PersistentFlags().BoolVar(&syncmediatrack.Verbose, "verbose", false, "Show more information")
	rootCmd.PersistentFlags().StringVar(&syncmediatrack.DefaultCountry, "defaultcountry", "", "Remove this country from geocoding")
	rootCmd.PersistentFlags().StringVar(&track, "track", "", "GPX track or a directory of GPX tracks")
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
