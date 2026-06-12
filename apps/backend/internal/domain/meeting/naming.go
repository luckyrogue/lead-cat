package meeting

import (
	"fmt"
	"time"
)

func GenerateName(dept, mtype, host string, date time.Time, r Recurrence) string {
	name := fmt.Sprintf("%s | %s | %s | %s", dept, mtype, host, date.Format("2006-01-02"))
	if label := r.Label(); label != "" {
		name += " | " + label
	}
	return name
}
