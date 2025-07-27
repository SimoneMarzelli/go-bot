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

var (
	UpdateFeed gtfs.FeedMessage
	FeedLock   sync.RWMutex

	StaticData     any
	StaticDataLock sync.RWMutex
)

func parse_feed() {
	FeedLock.Lock()
	defer FeedLock.Unlock()

	data, err := os.ReadFile(CURRENT_POSITION_URI)
	if err != nil {
		log.Fatal("Could not read feed file")
	}
	proto.Unmarshal(data, &UpdateFeed)
}

func parse_static() {
	StaticDataLock.Lock()
	defer StaticDataLock.Unlock()

	unzip_err := unzip(STATIC_DATA_URI, "./static")
	if unzip_err != nil {
		log.Fatal("Error unzipping")
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go parse_stop_names(&wg)
	go parse_directions(&wg)
	go parse_stop_times(&wg)

	wg.Wait()

}

var STOP_MAP map[string]string = make(map[string]string, 0)

func parse_stop_names(wg *sync.WaitGroup) {
	defer wg.Done()

	file, err := os.Open(STOPS_CSV_URI)
	if err != nil {
		log.Fatal("Could not read stops csv")
	}

	defer file.Close()

	reader := bufio.NewScanner(file)

	for reader.Scan() {
		line := reader.Text()
		split := strings.Split(line, ",")
		STOP_MAP[split[0]] = strings.ReplaceAll(split[2], "\"", "")
	}
}

var DIRECTION_MAP map[string][2]string = make(map[string][2]string, 0)

func parse_directions(wg *sync.WaitGroup) {
	defer wg.Done()

	file, err := os.Open(DIRECTIONS_CSV_URI)
	if err != nil {
		log.Fatal("Could not read directions csv")
	}

	defer file.Close()

	reader := bufio.NewScanner(file)

	for reader.Scan() {
		line := reader.Text()
		split := strings.Split(line, ",")

		route_id := split[0]
		dir_name := split[3]
		dir_id, _ := strconv.ParseUint(split[5], 10, 32)

		val := DIRECTION_MAP[route_id]
		val[dir_id] = strings.ReplaceAll(dir_name, "\"", "")
		DIRECTION_MAP[route_id] = val

	}
}

const STOP_TIMES_URI = "./static/stop_times.csv"

var STOP_TIMES_MAP map[string][]string = make(map[string][]string, 0)

func parse_stop_times(wg *sync.WaitGroup) {
	defer wg.Done()

	file, err := os.Open(STOP_TIMES_URI)
	if err != nil {
		log.Fatal("Could not read directions csv")
	}

	defer file.Close()

	reader := bufio.NewScanner(file)
	for reader.Scan() {
		line := reader.Text()
		split := strings.Split(line, ",")

		trip_id := split[0]
		stop_id := split[3]

		val := STOP_TIMES_MAP[trip_id]
		val = append(val, stop_id)
		STOP_TIMES_MAP[trip_id] = val
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
	ticker := time.NewTicker(interval)

	for {
		go download_data(url, out_file, parse)
		<-ticker.C
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
