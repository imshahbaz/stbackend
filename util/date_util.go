package util

import (
	"strings"
	"time"
)

var (
	inputLayout  = "02-Jan-2006"
	outputLayout = "2006-01-02"
)

func ParseNseDate(nseDate string) (string, error) {
	cleanInput := strings.TrimSpace(nseDate)
	t, err := time.Parse(inputLayout, cleanInput)
	if err != nil {
		return "", err
	}
	return t.Format(outputLayout), nil
}
