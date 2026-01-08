package util

import (
	"backend/model"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func Read(r io.Reader, minLeverage float32) ([]model.Margin, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, name := range header {
		headerMap[name] = i
	}

	symbolIdx, hasSymbol := headerMap["tradingsymbol"]
	leverageIdx, hasLeverage := headerMap["leverage"]
	if !hasSymbol || !hasLeverage {
		return nil, fmt.Errorf("missing required columns: tradingsymbol or leverage")
	}

	var margins []model.Margin

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			return nil, fmt.Errorf("error reading csv record: %w", err)
		}

		lev, err := strconv.ParseFloat(record[leverageIdx], 32)
		if err != nil {
			continue // Skip rows with invalid numbers
		}

		lev32 := float32(lev)

		if lev32 >= minLeverage {
			symbol := record[symbolIdx]
			margins = append(margins, model.Margin{
				Symbol: symbol,
				Name:   symbol,
				Margin: lev32,
			})
		}
	}

	return margins, nil
}

func ReadCSVReversed(r io.Reader, stopDate string) ([]model.ObRequest, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	var results []model.ObRequest

	for i := len(records) - 1; i >= 1; i-- {
		record := records[i]

		if len(record) < 2 {
			continue
		}

		rawDate := record[0] // e.g., "24-12-2025"
		symbol := record[1]  // e.g., "CHOLAFIN"

		parts := strings.Split(rawDate, "-")
		formattedDate := rawDate

		if formattedDate == stopDate {
			break
		}

		if len(parts) == 3 {
			formattedDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}

		results = append(results, model.ObRequest{
			Date:   formattedDate,
			Symbol: symbol,
		})
	}

	return results, nil
}
