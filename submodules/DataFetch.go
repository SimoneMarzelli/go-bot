package submodules

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

type UpdateFeed struct {
	updateFeed gtfs.FeedMessage
	lock       sync.Mutex
}

var FeedData = new(UpdateFeed)

type Static struct {
	stopMap          map[string]string
	tripIdToStopsMap map[string][]StopInfo
	lock             sync.Mutex
	totalMap         map[string]map[Direction]map[string][]StopInfo
}

var StaticData = &Static{
	stopMap:          make(map[string]string, 0),
	tripIdToStopsMap: make(map[string][]StopInfo, 0),
	totalMap:         make(map[string]map[Direction]map[string][]StopInfo),
}

const (
	CURRENT_POSITION_URI = "proto/rome_rtgtfs_vehicle_positions_feed.pb"
	CURRENT_POSITION_URL = "https://romamobilita.it/sites/default/files/rome_rtgtfs_vehicle_positions_feed.pb"

	STATIC_DATA_URL = "https://romamobilita.it/sites/default/files/rome_static_gtfs.zip"
	STATIC_DATA_URI = "static/rome_static_gtfs.zip"

	STOPS_URI      = "./static/stops.csv"
	TRIPS_URI      = "./static/trips.csv"
	STOP_TIMES_URI = "./static/stop_times.csv"
)

func parseFeed() {
	FeedData.lock.Lock()
	defer FeedData.lock.Unlock()

	data, err := os.ReadFile(CURRENT_POSITION_URI)
	if err != nil {
		log.Fatal("Could not read feed file")
	}
	proto.Unmarshal(data, &FeedData.updateFeed)
}

type Direction struct {
	id   string
	name string
}

func parseStatic() {
	StaticData.lock.Lock()
	defer StaticData.lock.Unlock()

	unzipErr := unzip(STATIC_DATA_URI, "./static")
	if unzipErr != nil {
		log.Fatal("Error unzipping")
	}

	parseStopNames()
	parseStopTimes()
	parseTrips()

}

func parseStopNames() {

	stops, err := os.Open(STOPS_URI)
	if err != nil {
		log.Fatal("could not read stops")
	}

	defer stops.Close()
	scanner := bufio.NewScanner(stops)

	for scanner.Scan() {
		split := strings.Split(scanner.Text(), ",")
		StaticData.stopMap[split[0]] = strings.ReplaceAll(split[2], "\"", "")
	}
}

type StopInfo struct {
	id          string
	name        string
	arrivalTime string
	sequence    uint64
}

func parseStopTimes() {

	file, err := os.Open(STOP_TIMES_URI)
	if err != nil {
		log.Fatal("Could not read directions csv")
	}

	defer file.Close()

	reader := bufio.NewScanner(file)
	for reader.Scan() {
		split := strings.Split(reader.Text(), ",")

		tripId := split[0]

		arrivalTime := split[1]
		stopId := split[3]
		sequence, _ := strconv.ParseUint(split[4], 10, 16)

		val := StaticData.tripIdToStopsMap[tripId]
		StaticData.tripIdToStopsMap[tripId] = append(val, StopInfo{
			id:          stopId,
			name:        StaticData.stopMap[stopId],
			arrivalTime: arrivalTime,
			sequence:    sequence,
		})
	}
}

func parseTrips() {
	tripsFile, err := os.Open(TRIPS_URI)
	if err != nil {
		log.Fatalln("Error reading trips")
	}

	defer tripsFile.Close()

	scanner := bufio.NewScanner(tripsFile)

	for scanner.Scan() {
		split := strings.Split(scanner.Text(), ",")
		routeId := split[0]

		tripId := split[2]
		directionName := strings.ReplaceAll(split[3], "\"", "")
		directionId := split[5]

		dirStruct := Direction{
			directionId,
			directionName,
		}

		_, ok := StaticData.totalMap[routeId]
		if !ok {
			StaticData.totalMap[routeId] = make(map[Direction]map[string][]StopInfo)
		}
		trips, ok := StaticData.totalMap[routeId][dirStruct]
		if !ok {
			StaticData.totalMap[routeId][dirStruct] = make(map[string][]StopInfo, 0)
		}
		prevStops, ok := trips[tripId]
		if !ok {
			StaticData.totalMap[routeId][dirStruct][tripId] = make([]StopInfo, 0)
		}
		StaticData.totalMap[routeId][dirStruct][tripId] = append(prevStops, StaticData.tripIdToStopsMap[tripId]...)
	}
}

// unzip files to static directory, converted to csv
func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		filePath := filepath.Join(dest, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(filePath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", filePath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		// Make sure directory exists
		if err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		if s, c := strings.CutSuffix(filePath, ".txt"); c {
			filePath = s + ".csv"
		}

		// Create destination file
		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func downloadData(url string, outFile string, parse func()) {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal("Error downloading update")
	}

	defer response.Body.Close()

	out, err := os.Create(outFile)
	if err != nil {
		log.Fatal("Could not fetch live data")
	}
	defer out.Close()

	_, err = io.Copy(out, response.Body)
	if err == nil {
		parse()
		log.Printf("Refreshed %v\n", outFile)
	}
}

func fetchRoutine(url string, outFile string, interval time.Duration, parse func()) {
	for {
		downloadData(url, outFile, parse)
		time.Sleep(interval)
	}
}

func StartFetching() {
	go fetchRoutine(
		CURRENT_POSITION_URL,
		CURRENT_POSITION_URI,
		60*time.Second,
		parseFeed,
	)

	go fetchRoutine(
		STATIC_DATA_URL,
		STATIC_DATA_URI,
		24*time.Hour,
		parseStatic,
	)
}
