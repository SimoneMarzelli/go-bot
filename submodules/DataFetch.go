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
	update_feed gtfs.FeedMessage
	lock        sync.Mutex
}

var FeedData = new(UpdateFeed)

type Static struct {
	stop_map             map[string]string
	trip_id_to_stops_map map[string][]StopInfo
	lock                 sync.Mutex
	total_map            map[string]map[Direction]map[string][]StopInfo
}

var StaticData = &Static{
	stop_map:             make(map[string]string, 0),
	trip_id_to_stops_map: make(map[string][]StopInfo, 0),
	total_map:            make(map[string]map[Direction]map[string][]StopInfo),
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

func parse_feed() {
	FeedData.lock.Lock()
	defer FeedData.lock.Unlock()

	data, err := os.ReadFile(CURRENT_POSITION_URI)
	if err != nil {
		log.Fatal("Could not read feed file")
	}
	proto.Unmarshal(data, &FeedData.update_feed)
}

type Direction struct {
	id   string
	name string
}

func parse_static() {
	StaticData.lock.Lock()
	defer StaticData.lock.Unlock()

	unzip_err := unzip(STATIC_DATA_URI, "./static")
	if unzip_err != nil {
		log.Fatal("Error unzipping")
	}

	parse_stop_names()

	parse_stop_times()
	parse_trips()

}

func parse_stop_names() {
	stops, err := os.Open(STOPS_URI)
	if err != nil {
		log.Fatal("could not read stops")
	}

	defer stops.Close()
	scanner := bufio.NewScanner(stops)

	for scanner.Scan() {
		split := strings.Split(scanner.Text(), ",")
		StaticData.stop_map[split[0]] = strings.ReplaceAll(split[2], "\"", "")
	}
}

func parse_trips() {
	trips_file, err := os.Open(TRIPS_URI)
	if err != nil {
		log.Fatalln("Error reading trips")
	}

	defer trips_file.Close()

	scanner := bufio.NewScanner(trips_file)

	for scanner.Scan() {
		split := strings.Split(scanner.Text(), ",")
		route_id := split[0]

		trip_id := split[2]
		direction_name := strings.ReplaceAll(split[3], "\"", "")
		direction_id := split[5]

		dir_struct := Direction{
			direction_id,
			direction_name,
		}

		_, ok := StaticData.total_map[route_id]
		if !ok {
			StaticData.total_map[route_id] = make(map[Direction]map[string][]StopInfo)
		}
		trips, ok := StaticData.total_map[route_id][dir_struct]
		if !ok {
			StaticData.total_map[route_id][dir_struct] = make(map[string][]StopInfo, 0)
		}
		prev_stops, ok := trips[trip_id]
		if !ok {
			StaticData.total_map[route_id][dir_struct][trip_id] = make([]StopInfo, 0)
		}
		StaticData.total_map[route_id][dir_struct][trip_id] = append(prev_stops, StaticData.trip_id_to_stops_map[trip_id]...)
	}
}

type StopInfo struct {
	id           string
	name         string
	arrival_time string
	sequence     uint64
}

func parse_stop_times() {

	file, err := os.Open(STOP_TIMES_URI)
	if err != nil {
		log.Fatal("Could not read directions csv")
	}

	defer file.Close()

	reader := bufio.NewScanner(file)
	for reader.Scan() {
		split := strings.Split(reader.Text(), ",")

		trip_id := split[0]

		arrival_time := split[1]
		stop_id := split[3]
		sequence, _ := strconv.ParseUint(split[4], 10, 16)

		val := StaticData.trip_id_to_stops_map[trip_id]
		StaticData.trip_id_to_stops_map[trip_id] = append(val, StopInfo{
			id:           stop_id,
			name:         StaticData.stop_map[stop_id],
			arrival_time: arrival_time,
			sequence:     sequence,
		})
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

func download_data(url string, out_file string, parse func()) {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal("Error downloading update")
	}

	defer response.Body.Close()

	out, err := os.Create(out_file)
	if err != nil {
		log.Fatal("Could not fetch live data")
	}
	defer out.Close()

	_, err = io.Copy(out, response.Body)
	if err == nil {
		parse()
		log.Printf("Refreshed %v\n", out_file)
	}
}

func fetch_routine(url string, out_file string, interval time.Duration, parse func()) {
	for {
		download_data(url, out_file, parse)
		time.Sleep(interval)
	}
}

func StartFetching() {
	go fetch_routine(
		CURRENT_POSITION_URL,
		CURRENT_POSITION_URI,
		60*time.Second,
		parse_feed,
	)

	go fetch_routine(
		STATIC_DATA_URL,
		STATIC_DATA_URI,
		24*time.Hour,
		parse_static,
	)
}
