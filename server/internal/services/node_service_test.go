package services

import (
	"fmt"
	"testing"

	"github.com/sanda0/vps_pilot/internal/dto"
)

// Every time range button in the dashboard must reach the query as the right
// number of seconds.
func TestTimeRangeButtonsToSeconds(t *testing.T) {
	for button, want := range map[string]int64{
		"5M":  5 * 60,
		"15M": 15 * 60,
		"1H":  3600,
		"1D":  86400,
		"2D":  2 * 86400,
		"7D":  7 * 86400,
		"1W":  7 * 86400,
	} {
		var req dto.NodeSystemStatRequestDto
		if err := req.FromBytes([]byte(fmt.Sprintf(`{"id":1,"time_range":%q}`, button))); err != nil {
			t.Fatal(err)
		}
		if got := parseTimeRangeToSeconds(req.TimeRange); got != want {
			t.Errorf("%s: got %d seconds, want %d", button, got, want)
		}
	}
}
