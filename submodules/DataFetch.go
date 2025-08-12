package submodules

import (
	"bufio"
	"errors"
	"go-bot/utils"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

type CurrentPosition struct {
	currentPositionFeed gtfs.FeedMessage
	lock                sync.Mutex
}

var PositionData = new(CurrentPosition)

type Updates struct {
	updateFeed gtfs.FeedMessage
	lock       sync.Mutex
}

var UpdateData = new(Updates)

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
	CurrentPositionUri = "proto/rome_rtgtfs_vehicle_positions_feed.pb"
	CurrentPositionUrl = "https://romamobilita.it/sites/default/files/rome_rtgtfs_vehicle_positions_feed.pb"

	UpdatesUri = "proto/rome_rtgtfs_trip_updates_feed.pb"
	UpdatesUrl = "https://romamobilita.it/sites/default/files/rome_rtgtfs_trip_updates_feed.pb"

	StaticDataUrl    = "https://romamobilita.it/sites/default/files/rome_static_gtfs.zip"
	StaticDataUri    = "static/rome_static_gtfs.zip"
	StaticDataMD5Url = "https://romamobilita.it/sites/default/files/rome_static_gtfs.zip.md5"

	StopsUri     = "./static/stops.csv"
	TripsUri     = "./static/trips.csv"
	StopTimesUri = "./static/stop_times.csv"
)

func parsePositions() {
	PositionData.lock.Lock()
	defer PositionData.lock.Unlock()

	data, err := os.ReadFile(CurrentPositionUri)
	if err != nil {
		log.Fatal("Could not read positions file")
	}

	err = proto.Unmarshal(data, &PositionData.currentPositionFeed)
	if err != nil {
		log.Fatalf("Could not parse positions file: %v", err)
	}
}

func parseUpdates() {
	UpdateData.lock.Lock()
	defer UpdateData.lock.Unlock()

	data, err := os.ReadFile(UpdatesUri)
	if err != nil {
		log.Fatal("Could not read update file")
	}
	err = proto.Unmarshal(data, &UpdateData.updateFeed)
	if err != nil {
		log.Fatalf("Could not parse update file: %v", err)
	}
}

type Direction struct {
	id   string
	name string
}

func parseStatic() {
	StaticData.lock.Lock()
	defer StaticData.lock.Unlock()

	unzipErr := utils.Unzip(StaticDataUri, "./static")
	if unzipErr != nil {
		log.Fatal("Error unzipping")
	}

	parseStopNames()
	parseStopTimes()
	parseTrips()

}

func parseStopNames() {

	stops, err := os.Open(StopsUri)
	if err != nil {
		log.Fatal("could not read stops")
	}

	defer utils.HandleFileClose(stops)

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

	file, err := os.Open(StopTimesUri)
	if err != nil {
		log.Fatal("Could not read directions csv")
	}

	defer utils.HandleFileClose(file)

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
	tripsFile, err := os.Open(TripsUri)
	if err != nil {
		log.Fatalln("Error reading trips")
	}

	defer utils.HandleFileClose(tripsFile)

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

func fetchRoutine(
	url string,
	outFile string,
	interval time.Duration,
	refresh func() bool,
	parse func()) {
	ticker := time.NewTicker(interval)
	for {
		var err error = nil
		if refresh() {
			log.Printf("File not updated, redownloading %v...", outFile)
			err = utils.DownloadFile(url, outFile)
			if err != nil {
				continue
			}
		}

		log.Printf("File updated, parse %v...", outFile)
		parse()
		log.Printf("Refreshed %v\n", outFile)

		<-ticker.C
	}
}

func initDirs() {
	err := os.Mkdir("proto", os.ModePerm)
	if err != nil && !errors.Is(err, os.ErrExist) {
		log.Fatalf("Error creating proto directory: %v", err)
	}

	err = os.Mkdir("static", os.ModePerm)
	if err != nil && !errors.Is(err, os.ErrExist) {
		log.Fatalf("Error creating static directory: %v", err)
	}
}

func StartFetching() {

	initDirs()

	go fetchRoutine(
		CurrentPositionUrl,
		CurrentPositionUri,
		60*time.Second,
		func() bool {
			return true
		},
		parsePositions,
	)

	go fetchRoutine(
		UpdatesUrl,
		UpdatesUri,
		60*time.Second,
		func() bool {
			return true
		},
		parseUpdates,
	)

	fetchRoutine(
		StaticDataUrl,
		StaticDataUri,
		24*time.Hour,
		func() bool {
			hash, err := utils.ReadRemoteFile(StaticDataMD5Url)
			if err != nil {
				log.Println("Error reading remote file")
				return true
			}

			hashFile, err := os.OpenFile("static/old_hash", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			defer utils.HandleFileClose(hashFile)

			if err != nil {
				log.Println("Error reading old hash")
				return true
			}

			oldHash := make([]byte, 16)
			_, err = hashFile.Read(oldHash)
			if err != nil {
				return true
			}

			if string(oldHash) != hash {
				_, err := hashFile.Write([]byte(hash))
				if err != nil {
					log.Fatalf("Could not write to hash file: %v", err)
				}

				return true
			}

			return false
		},
		parseStatic,
	)
}
