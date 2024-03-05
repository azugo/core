package time

import (
	"fmt"
	"time"
)

// Time is a wrapper around time.Time that supports JSON marshaling.
type Time time.Time

func (t *Time) String() string {
	if t == nil || time.Time(*t).IsZero() {
		return ""
	}

	return time.Time(*t).Format(time.RFC3339Nano)
}

func (t *Time) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte(""), nil
	}

	return []byte(`"` + time.Time(*t).Format(time.RFC3339Nano) + `"`), nil
}

func (t *Time) UnmarshalJSON(data []byte) error {
	tt, err := time.Parse(`"`+time.RFC3339Nano+`"`, string(data))
	if err != nil {
		return err
	}

	*t = Time(tt)

	return nil
}

// Date is a wrapper around time.Time that supports JSON marshaling to date.
type Date time.Time

func (t *Date) String() string {
	if t == nil || time.Time(*t).IsZero() {
		return ""
	}

	y, m, d := time.Time(*t).Date()
	return fmt.Sprintf("%04d-%02d-%02d", y, int(m), d)
}

func (t *Date) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte(""), nil
	}

	return []byte(time.Time(*t).Format(`"2006-01-02"`)), nil
}

func (t *Date) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	tt, err := time.Parse(`"2006-01-02"`, string(data))
	if err != nil {
		// If parsing as pure date fails, try parsing as timestamp
		tt, err = time.Parse(fmt.Sprintf(`"%s"`, time.RFC3339Nano), string(data))
		if err != nil {
			return err
		}
	}

	*t = Date(tt)

	return nil
}
