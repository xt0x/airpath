package domain

import (
	"errors"
	"regexp"
	"time"
)

type NormalizeLocalDateTimeInput struct {
	LocalDateTime string
	TimeZone      string
}

var utcISODateTimePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$`)

func NormalizeUTCISODateTime(value string) (ISODateTimeString, error) {
	if !utcISODateTimePattern.MatchString(value) {
		return "", errors.New("invalid ISO 8601 date-time with timezone")
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return "", errors.New("invalid ISO 8601 date-time with timezone")
	}

	return ISODateTimeString(parsed.UTC().Format("2006-01-02T15:04:05Z")), nil
}

func NormalizeLocalDateTimeToUTCISO(input NormalizeLocalDateTimeInput) (ISODateTimeString, error) {
	location, err := time.LoadLocation(input.TimeZone)
	if err != nil {
		return "", errors.New("invalid IANA timezone")
	}

	parsed, err := parseLocalDateTimeInLocation(input.LocalDateTime, location)
	if err != nil {
		return "", errors.New("invalid local ISO 8601 date-time")
	}

	return ISODateTimeString(parsed.UTC().Format("2006-01-02T15:04:05Z")), nil
}

func parseLocalDateTimeInLocation(value string, location *time.Location) (time.Time, error) {
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04"} {
		parsed, err := time.ParseInLocation(layout, value, location)
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, errors.New("invalid local ISO 8601 date-time")
}
