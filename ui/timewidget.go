package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2/widget"
)

const (
	TimestampFormat = "15:04"
)

type TimeEntry struct {
	*widget.Entry
	Timestamp time.Time
}

func CreateTimeEntry() *TimeEntry {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("HH:MM")

	result := &TimeEntry{
		Entry:     entry,
		Timestamp: time.Now(),
	}
	entry.SetText(result.Timestamp.Format(TimestampFormat))

	entry.Validator = func(s string) error {
		if _, err := time.Parse(TimestampFormat, s); err != nil {
			return fmt.Errorf("%q ist keine valide Uhrzeit!", s)
		}
		return nil
	}
	entry.OnChanged = func(s string) {
		time, err := time.Parse(TimestampFormat, s)
		if err == nil {
			result.Timestamp = time
			return
		}
		if len(s) >= 3 && s[2] != ':' {
			entry.SetText(s[:2] + ":" + s[2:])
			entry.CursorColumn = entry.CursorColumn + 1
		}
		if len(s) > 5 {
			entry.SetText(s[:5])
		}
	}

	return result
}
