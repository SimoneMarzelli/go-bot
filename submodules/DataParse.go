package submodules

import (
	"fmt"
	"strings"
)

type StopUpdate struct {
	Name         string
	Status       []string
	ArrivalTimes []string
}

func GetCurrentPosition(route_id string, direction string) ([]StopUpdate, error) {
	var update_feed = FeedData
	update_feed.lock.Lock()
	defer update_feed.lock.Unlock()

	var staticData = StaticData
	staticData.lock.Lock()
	defer staticData.lock.Unlock()

	directions_map, ok := staticData.total_map[route_id]
	if !ok {
		return nil, fmt.Errorf("route non recognized")
	}

	var trips map[string][]StopInfo

	for key_dir, v := range directions_map {
		if key_dir.id == direction || strings.Contains(strings.ToLower(key_dir.name), strings.ToLower(direction)) {
			trips = v
			break
		}
	}

	if len(trips) == 0 {
		return nil, fmt.Errorf("unrecognized direction")
	}

	var ret []StopUpdate
	for _, entity := range update_feed.update_feed.Entity {
		vehicle_info := entity.Vehicle

		if vehicle_info.Trip.GetRouteId() != route_id {
			continue
		}

		trip_id := vehicle_info.Trip.GetTripId()
		current_status := vehicle_info.GetCurrentStatus()
		current_stop := staticData.stop_map[vehicle_info.GetStopId()]

		trip_stops := trips[trip_id]

		for idx, trip_stop := range trip_stops {
			if idx == len(ret) {
				ret = append(ret, StopUpdate{
					Name:         trip_stop.name,
					ArrivalTimes: []string{},
					Status:       []string{},
				})
			}

			ret[idx].ArrivalTimes = append(ret[idx].ArrivalTimes, trip_stop.arrival_time)
			if trip_stop.name == current_stop {
				ret[idx].Status = append(ret[idx].Status, current_status.String())
			}
		}

	}
	return ret, nil
}

func GetLineInfo(route_id string) ([]string, error) {
	var staticData = StaticData
	staticData.lock.Lock()
	defer staticData.lock.Unlock()

	directions, ok := staticData.total_map[route_id]
	if ok {

		keys := make([]string, len(directions))

		i := 0
		for k := range directions {
			keys[i] = k.name
			i++
		}

		return keys, nil
	}

	return nil, fmt.Errorf("route %v does not exist", route_id)
}
