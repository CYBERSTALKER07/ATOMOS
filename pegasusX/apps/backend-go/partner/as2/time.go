package as2

import "time"

func timeUnixNano() int64 {
	return time.Now().UnixNano()
}
