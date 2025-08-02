package datetime

import (
	"fmt"
	"time"
)

func StrToTimeWithMicro(dateTime string) (int64, error) {
	t, err := time.Parse("2006-01-02T15:04:05.000Z", dateTime)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05", dateTime)
		if err != nil {
			return 0, err
		}
	}

	secs := t.Unix()
	micros := t.Nanosecond() / 1000

	resultStr := fmt.Sprintf("%d%06d", secs, micros)

	var result int64
	fmt.Sscanf(resultStr, "%d", &result)
	return result, nil
}
