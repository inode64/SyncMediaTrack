package cmd

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	syncmediatrack "github.com/inode64/SyncMediaTrack/lib"
	"github.com/karrick/godirwalk"
	"github.com/spf13/cobra"
)

type ImageInfo struct {
	Path         string
	atime        time.Time
	etime        time.Time
	gtime        time.Time
	HasGPSDate   bool
	AdjustedDate time.Time
	IsAdjusted   bool
}

type Segment struct {
	Diff      time.Duration
	StartTime time.Time
	EndTime   time.Time
	Id        []string
}

var MaxTimeSegment float64 = 60 * 4 // 4 hours
var imageFile = map[string][]ImageInfo{}
var DenyExtension = []string{"LRV", "THM"}

var fixTimeCmd = &cobra.Command{
	Use:   "fixtime",
	Short: "Fix time in image files",
	Long:  `Corrects time in image files using GEO data and other adjacent images to correct time shift`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			mediaDir = args[0]
		}
		fixTimeExecute()
	},
}

func GetTime(atime time.Time, etime time.Time, gtime time.Time) time.Time {
	if !gtime.IsZero() {
		return gtime
	}
	if !etime.IsZero() {
		return gtime
	}
	return atime
}

func init() {
	rootCmd.AddCommand(fixTimeCmd)

	imageFile = make(map[string][]ImageInfo)
}

func fixTimeExecute() {
	syncmediatrack.Pass("Reading medias...")
	syncmediatrack.Pass("First pass...")

	re := regexp.MustCompile(`([A-Za-z-_]*)(\d[\w.-]*)(\.[\w]+)$`)
	var split []string
	var id string
	var src time.Time

	err := godirwalk.Walk(mediaDir, &godirwalk.Options{
		Callback: func(path string, de *godirwalk.Dirent) error {
			var gpsOld syncmediatrack.Trkpt

			if de.IsDir() {
				return nil // do not remove directory that was provided top-level directory
			}

			if !syncmediatrack.FileIsMedia(path) {
				return nil
			}

			relPath, err := filepath.Rel(mediaDir, path)
			if err != nil {
				mediaError++
				return err
			}

			// Exclude extension to analyze
			for _, ext := range DenyExtension {
				if filepath.Ext(path) == "."+ext {
					return nil
				}
			}

			atime, etime, gtime, err := syncmediatrack.GetMediaDate(path, &gpsOld)
			if err != nil {
				mediaError++
				fmt.Println(err)
				return nil
			}

			split = re.FindStringSubmatch(relPath)
			mediaValid++
			fmt.Printf("[%v] - ", relPath)

			if len(split) < 2 {
				mediaError++
				fmt.Println(" - Error: Can't get file ID")
				return nil
			}

			id = split[2]
			// If the file start with GL o GX (GOPRO Files) remove 2 first characters from id
			if split[1] == "GL" || split[1] == "GX" {
				id = id[2:]
			}

			fmt.Printf(" ID: %s A: %s E: %s G: %s",
				id,
				atime.Format("02/01/2006 15:04:05"),
				etime.Format("02/01/2006 15:04:05"),
				gtime.Format("02/01/2006 15:04:05"),
			)

			if !etime.IsZero() {
				src = etime
			} else {
				src = atime
			}

			if !gtime.IsZero() {
				// diff times and adjust
				diff := gtime.Sub(src)
				fmt.Printf(" Diff: %s", diff.String())
			}

			fmt.Println()

			imageFile[id] = append(imageFile[id], ImageInfo{Path: path, atime: atime, etime: etime, gtime: gtime})

			mediaUpdate++

			return nil
		},
		Unsorted: false,
	})
	if err != nil {
		syncmediatrack.Warning(err.Error())
	}

	syncmediatrack.Pass("Second pass...")

	var Ids []string
	for k := range imageFile {
		Ids = append(Ids, k)
	}

	sort.Strings(Ids)

	for _, k := range Ids {
		fmt.Printf("Clave: %s\n", k)
		for _, value1 := range imageFile[k] {
			// Show path and date from ImageInfo
			fmt.Printf("Path: %s\n", value1.Path)
		}
	}

	seg := []Segment{}
	oldTime := time.Time{}
	Id := []string{}
	StoredTime := time.Time{}
	StartTime := time.Time{}
	EndTime := time.Time{}
	Diff := time.Duration(0)

	for _, k := range Ids {
		StoredTime = GetTime(imageFile[k][0].atime, imageFile[k][0].etime, imageFile[k][0].etime)
		if (oldTime != time.Time{}) {
			t := StoredTime.Sub(oldTime)
			if math.Abs(t.Minutes()) > MaxTimeSegment || oldTime.After(StoredTime) {
				seg = append(seg, Segment{Diff: Diff, StartTime: StartTime, EndTime: EndTime, Id: Id})
				Id = []string{}
				oldTime = time.Time{}
				StartTime = StoredTime
			}
		} else {
			StartTime = StoredTime
		}
		oldTime = StoredTime
		Id = append(Id, k)
		if !imageFile[k][0].gtime.IsZero() {
			Diff1 := StoredTime.Sub(imageFile[k][0].gtime)
			Diff = (Diff + Diff1) / 2
		}
	}
	if len(Id) > 0 {
		seg = append(seg, Segment{Diff: Diff, StartTime: StartTime, EndTime: EndTime, Id: Id})
	}

	for key, value := range seg {
		fmt.Printf("Segment: %d\n", key)
		fmt.Printf("Start: %s\n", value.StartTime.Format("02/01/2006 15:04:05"))
		fmt.Printf("End: %s\n", value.EndTime.Format("02/01/2006 15:04:05"))
		fmt.Printf("Diff: %s\n", value.Diff.String())
		for _, value1 := range value.Id {
			fmt.Printf("Id: %s\n", value1)
		}
	}
}
