package instorage

import (
	"strings"
)

//CLIRow maps headers to values in a row of a CLI cmd output
type CLIRow map[string]string

//ICLIParser is a interface that is used to parse the command output
//transform origianl data from raw string into structure data
type ICLIParser interface {
	Parse(rawData string, withHeader bool, delimiter string) []CLIRow
}

//CLIParser is the Parser to parse InStorage CLI output
type CLIParser struct{}

//NewCLIParser create a CLIParser object used to parse CLI output
func NewCLIParser() *CLIParser {
	return &CLIParser{}
}

//Parse just transform rawData into a list of map which map header name to value
func (p *CLIParser) Parse(rawData string, withHeader bool, delimiter string) []CLIRow {
	trimedOutput := strings.Trim(rawData, "\n")
	var lines = strings.Split(trimedOutput, "\n")

	var rows []CLIRow
	if withHeader {
		// the result is a table with first line as header row
		var headers []string
		for idxLine, line := range lines {
			if len(line) == 0 {
				break
			}

			columns := strings.Split(line, delimiter)
			//the result is a table with header row
			if idxLine == 0 {
				headers = columns
			} else {
				row := make(CLIRow)
				for idxHeader, header := range headers {
					row[header] = columns[idxHeader]
				}
				rows = append(rows, row)
			}
		}
	} else {
		// the result is a two column table with first colume as header
		// and the second column as value
		row := make(CLIRow)
		for _, line := range lines {
			if len(line) != 0 {
				columns := strings.Split(line, delimiter)
				row[columns[0]] = strings.Join(columns[1:], " ")
			} else {
				rows = append(rows, row)
				row = make(CLIRow)
			}
		}

		if len(row) != 0 {
			rows = append(rows, row)
		}
	}

	return rows
}

//Filter use to filter original CLIRow with filter parameter to a new CLIRow
func (p *CLIParser) Filter(rows []CLIRow, filters []string) []CLIRow {
	var newRows []CLIRow

	for _, row := range rows {
		newRow := make(CLIRow)
		for _, header := range filters {
			newRow[header] = row[header]
		}

		newRows = append(newRows, newRow)
	}

	return newRows
}
