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

func GetCurrentPosition(routeId string, direction string) ([]StopUpdate, error) {
	var updateFeed = FeedData
	updateFeed.lock.Lock()
	defer updateFeed.lock.Unlock()

	var staticData = StaticData
	staticData.lock.Lock()
	defer staticData.lock.Unlock()

	directionsMap, ok := staticData.totalMap[routeId]
	if !ok {
		return nil, fmt.Errorf("unrecognized route")
	}

	var trips map[string][]StopInfo

	for keyDir, v := range directionsMap {
		if keyDir.id == direction || strings.Contains(strings.ToLower(keyDir.name), strings.ToLower(direction)) {
			trips = v
			break
		}
	}

	if len(trips) == 0 {
		return nil, fmt.Errorf("unrecognized direction")
	}

	var ret []StopUpdate
	for _, entity := range updateFeed.updateFeed.Entity {
		vehicleInfo := entity.Vehicle

		if vehicleInfo.Trip.GetRouteId() != routeId {
			continue
		}

		tripId := vehicleInfo.Trip.GetTripId()
		currentStatus := vehicleInfo.GetCurrentStatus()
		currentStop := staticData.stopMap[vehicleInfo.GetStopId()]

		tripStops := trips[tripId]

		for idx, tripStop := range tripStops {
			if idx == len(ret) {
				ret = append(ret, StopUpdate{
					Name:         tripStop.name,
					ArrivalTimes: []string{},
					Status:       []string{},
				})
			}

			ret[idx].ArrivalTimes = append(ret[idx].ArrivalTimes, tripStop.arrivalTime)
			if tripStop.name == currentStop {
				ret[idx].Status = append(ret[idx].Status, currentStatus.String())
			}
		}

	}

	return ret, nil
}

func GetLineInfo(routeId string) ([]string, error) {
	var staticData = StaticData
	staticData.lock.Lock()
	defer staticData.lock.Unlock()

	directions, ok := staticData.totalMap[routeId]
	if ok {

		keys := make([]string, len(directions))

		i := 0
		for k := range directions {
			keys[i] = k.name
			i++
		}

		return keys, nil
	}

	return nil, fmt.Errorf("route %v does not exist", routeId)
}
