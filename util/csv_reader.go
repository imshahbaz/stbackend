package util

import (
	"backend/model"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Read handles the CSV parsing logic
// We use io.Reader so it works with file uploads, local files, or strings
func Read(r io.Reader, minLeverage float32) ([]model.Margin, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	// 1. Read the Header row
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Create a map to find column indices by name (mimicking record.get("name"))
	headerMap := make(map[string]int)
	for i, name := range header {
		headerMap[name] = i
	}

	// Validate required columns exist
	symbolIdx, hasSymbol := headerMap["tradingsymbol"]
	leverageIdx, hasLeverage := headerMap["leverage"]
	if !hasSymbol || !hasLeverage {
		return nil, fmt.Errorf("missing required columns: tradingsymbol or leverage")
	}

	var margins []model.Margin

	// 2. Iterate through records
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			return nil, fmt.Errorf("error reading csv record: %w", err)
		}

		// Parse leverage string to float32
		lev, err := strconv.ParseFloat(record[leverageIdx], 32)
		if err != nil {
			continue // Skip rows with invalid numbers
		}

		lev32 := float32(lev)

		// 3. Filter and Build (Builder pattern replaced by struct literal)
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

// ReadCSVReversed reads from a reader, iterates from the last line,
// and returns date (YYYY-MM-DD) and symbol.
func ReadCSVReversed(r io.Reader, stopDate string) ([]model.ObRequest, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	// Read all records to allow backward iteration
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	var results []model.ObRequest

	// Start reading from the last record to the first
	for i := len(records) - 1; i >= 1; i-- {
		record := records[i]

		// Basic validation: ensure we have at least date and symbol
		if len(record) < 2 {
			continue
		}

		rawDate := record[0] // e.g., "24-12-2025"
		symbol := record[1]  // e.g., "CHOLAFIN"

		// Reformat Date: DD-MM-YYYY -> YYYY-MM-DD
		parts := strings.Split(rawDate, "-")
		formattedDate := rawDate

		if formattedDate == stopDate {
			break
		}

		if len(parts) == 3 {
			// Rearrange to YYYY-MM-DD
			formattedDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}

		results = append(results, model.ObRequest{
			Date:   formattedDate,
			Symbol: symbol,
		})
	}

	return results, nil
}
