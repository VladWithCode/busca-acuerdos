package tsj

import (
	"fmt"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/vladwithcode/juzgados/internal/db"
	"github.com/vladwithcode/juzgados/internal/reader"
)

var (
	IDX_LEN    = 7
	CASE_LEN   = 15
	NATURE_LEN = 23
	ACCORD_LEN = 49
)

type NotFoundError struct {
	Message string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("[NotFound error] %s\n", e.Message)
}

func GetCaseData(caseId, caseType string, searchDate *time.Time, daysBack int) (*db.Doc, error) {
	if searchDate == nil {
		t := time.Now()
		searchDate = &t
	}

	var data []byte
	var err error

	for i := 0; i <= daysBack; i++ {
		y, m, d := searchDate.Date()
		date := fmt.Sprintf("%d%d%d", d, m, y)
		data, err = FetchAndReadDoc(caseId, date, caseType)

		if data != nil {
			break
		}

		t := searchDate.AddDate(0, 0, -1)
		searchDate = &t

		if i < daysBack {
			err = nil
		}
	}

	if err != nil {
		return nil, err
	}

	doc := DataToDoc(data)

	doc.AccordDate = *searchDate
	doc.NatureCode = caseType

	return doc, nil
}

func FetchAndReadDoc(caseId, searchDate, caseType string) ([]byte, error) {
	pdfContent, err := reader.Reader(searchDate, caseType)

	if err != nil {
		fmt.Println(err)

		return nil, err
	}

	searchExp, err := reader.GenRegExp(caseId)

	if err != nil {
		fmt.Println(err)

		return nil, err
	}

	idx := searchExp.FindIndex(*pdfContent)

	if len(idx) == 0 {
		err := NotFoundError{
			Message: "No se encontró información sobre el caso solicitado",
		}

		return nil, err
	}

	start, end := idx[0], idx[1]

	type successResponse struct {
		Index   string `json:"index"`
		Content string `json:"content"`
	}

	contentAsBytes := (*pdfContent)[start:end]

	return contentAsBytes, nil
}

func DataToDoc(data []byte) *db.Doc {
	lineExp := regexp.MustCompile("(?m)\n")
	rows := lineExp.Split(string(data), -1)

	doc := db.Doc{}
	doc.ID = uuid.New().String()

	var cols [4][]byte
	var tempCols [4][]byte
	var currentCol int
	var seenTwoSpace bool
	var prevChar byte
	charCounts := []int{0, 0, 0, ACCORD_LEN}

	for i, str := range rows {
		currentCol = 0
		seenTwoSpace = false
		prevChar = 0
		tempCols[0] = []byte{}
		tempCols[1] = []byte{}
		tempCols[2] = []byte{}
		tempCols[3] = []byte{}

		for j, char := range str {
			if char == '\n' {
				break
			}

			if seenTwoSpace && char != ' ' && currentCol < 3 {
				currentCol++
				seenTwoSpace = false
				prevChar = 0
			}

			tempCols[currentCol] = utf8.AppendRune(tempCols[currentCol], char)

			// Keep track of the length of the columns for splitting following rows
			if i == 0 && currentCol < 3 {
				charCounts[currentCol]++
			}

			if char == ' ' && prevChar == ' ' {
				seenTwoSpace = true
			}

			colLen := len(tempCols[currentCol])
			maxLen := getColMaxLength(currentCol)

			if char == ' ' && str[ensureSafeIndex(j+1, len(str))] != ' ' && colLen >= maxLen && currentCol < 3 {
				currentCol++
				seenTwoSpace = false
				prevChar = 0
				continue
			}

			// Check if the length of the current column is at the max length
			if i > 0 && colLen == charCounts[currentCol] {
				// if true increment the col number
				currentCol++
				seenTwoSpace = false
				prevChar = 0
			}

			prevChar = byte(char)
		}

		cols[0] = append(cols[0], tempCols[0]...)
		cols[1] = append(cols[1], tempCols[1]...)
		cols[2] = append(cols[2], tempCols[2]...)
		cols[3] = append(cols[3], tempCols[3]...)
	}

	doc.Case = string(cols[1])
	doc.Nature = string(cols[2])
	doc.Accord = string(cols[3])

	return &doc
}

func ensureSafeIndex(idx, ln int) int {
	if idx < 0 {
		return 0
	}
	if idx > ln {
		return ln
	}

	return idx
}

func getColMaxLength(colNum int) int {
	switch colNum {
	case 0:
		return IDX_LEN
	case 1:
		return CASE_LEN
	case 2:
		return NATURE_LEN
	case 3:
		return ACCORD_LEN
	default:
		return 0
	}
}

/**
 * For some cases the space between cols it's 1 space char
 * Use special col splitting for those cases
 */
func handleColsWithOneSpace(data []byte, doc *db.Doc) *db.Doc {
	doc.FullText = string(data)

	return doc
}
