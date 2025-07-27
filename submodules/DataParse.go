package submodules

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

func GetCurrentPosition(route_id string, direction string) ([]string, map[string][]string, error) {

	data, _ := os.ReadFile(CURRENT_POSITION_URI)
	var feed gtfs.FeedMessage
	proto.Unmarshal(data, &feed)

	var current_stops map[string][]string = make(map[string][]string, 0)
	var ordered_stop_names []string = make([]string, 0)

	for _, entity := range feed.Entity {
		vehicle_info := entity.Vehicle

		if vehicle_info.Trip.GetRouteId() != route_id {
			continue
		}

		direction_id := uint64(vehicle_info.Trip.GetDirectionId())
		direction_name := DIRECTION_MAP[route_id][direction_id]
		if direction != strconv.FormatUint(direction_id, 10) || !strings.Contains(strings.ToLower(direction_name), strings.ToLower(direction_name)) {
			continue
		}

		trip_stops := STOP_TIMES_MAP[vehicle_info.Trip.GetTripId()]

		current_stop_name := STOP_MAP[vehicle_info.GetStopId()]
		current_stop_status := vehicle_info.GetCurrentStatus()

		for _, stop_id := range trip_stops {
			stop_name := STOP_MAP[stop_id]

			if len(ordered_stop_names) != len(trip_stops) {
				ordered_stop_names = append(ordered_stop_names, stop_name)
			}

			stop_info, ok := current_stops[stop_name]

			if !ok {
				current_stops[stop_name] = []string{}
			}

			if current_stop_name == stop_name {
				stop_info = append(stop_info, current_stop_status.String())
				current_stops[stop_name] = stop_info
			}
		}
	}
	fmt.Println(ordered_stop_names)
	return ordered_stop_names, current_stops, nil
}

func GetLineInfo(route_id string) ([]string, error) {
	directions, ok := DIRECTION_MAP[route_id]
	if ok {
		return directions[:], nil
	}

	return nil, fmt.Errorf("Route %v does not exist", route_id)
}
