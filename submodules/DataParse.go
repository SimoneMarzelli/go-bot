package submodules

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

func GetLineInfo(route_id string) (string, error) {

	data, _ := os.ReadFile(CURRENT_POSITION_URI)
	var feed gtfs.FeedMessage
	proto.Unmarshal(data, &feed)

	var ret strings.Builder

	for _, entity := range feed.Entity {
		vehicle_info := entity.Vehicle

		if strings.Compare(vehicle_info.Trip.GetRouteId(), route_id) != 0 {
			continue
		}

		stop_name := STOP_MAP[vehicle_info.GetStopId()]
		direction_name := DIRECTION_MAP[route_id][vehicle_info.Trip.GetDirectionId()]
		ret.WriteString(fmt.Sprintf(
			"%v stop %v, destination %v\n\n",
			strings.Map(
				func(r rune) rune {
					if r == '_' {
						return ' '
					} else {
						return unicode.ToLower(r)
					}
				},
				vehicle_info.GetCurrentStatus().String(),
			),
			stop_name,
			direction_name,
		))

	}

	return ret.String(), nil
}
