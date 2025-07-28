package submodules

import (
	"fmt"
	"strconv"
	"strings"
)

func GetCurrentPosition(route_id string, direction string) ([]string, map[string][]string, error) {
	var update_feed = FeedData
	update_feed.lock.Lock()
	defer update_feed.lock.Unlock()

	var staticData = StaticData
	staticData.lock.Lock()
	defer staticData.lock.Unlock()

	var current_stops map[string][]string = make(map[string][]string, 0)
	var ordered_stop_names []string = make([]string, 0)

	for _, entity := range update_feed.update_feed.Entity {
		vehicle_info := entity.Vehicle

		if vehicle_info.Trip.GetRouteId() != route_id {
			continue
		}

		direction_id := uint64(vehicle_info.Trip.GetDirectionId())
		direction_name := staticData.direction_map[route_id][direction_id]
		if direction != strconv.FormatUint(direction_id, 10) && !strings.Contains(strings.ToLower(direction_name), strings.ToLower(direction)) {
			continue
		}

		trip_stops := staticData.stop_times_map[vehicle_info.Trip.GetTripId()]

		current_stop_name := staticData.stop_map[vehicle_info.GetStopId()]
		current_stop_status := vehicle_info.GetCurrentStatus()

		for _, stop_id := range trip_stops {
			stop_name := staticData.stop_map[stop_id]

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

	return ordered_stop_names, current_stops, nil
}

func GetLineInfo(route_id string) ([]string, error) {
	var staticData = StaticData
	staticData.lock.Lock()
	defer staticData.lock.Unlock()

	directions, ok := staticData.direction_map[route_id]
	if ok {
		return directions[:], nil
	}

	return nil, fmt.Errorf("route %v does not exist", route_id)
}
