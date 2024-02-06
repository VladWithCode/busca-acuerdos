package tsj

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	db "github.com/vladwithcode/juzgados/internal/db/docs"
	"github.com/vladwithcode/juzgados/internal/reader"
)

var (
	IDX_LEN    = 7
	CASE_LEN   = 15
	NATURE_LEN = 23
	ACCORD_LEN = 49
)

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
		return nil, errors.New("No se encontró información sobre el caso solicitado")
	}

	start, end := idx[0], idx[1]

	type successResponse struct {
		Index   string `json:"index"`
		Content string `json:"content"`
	}

	contentAsStr := (*pdfContent)[start:end]

	return contentAsStr, nil
}

func DataToDoc(data []byte) *db.Doc {
	lineExp := regexp.MustCompile("(?m)\n")
	// trimExp := regexp.MustCompile("(?m)^ *| *$")
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

			tempCols[currentCol] = append(tempCols[currentCol], byte(char))

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

			// fmt.Println("Col 0", string(tempCols[0]))
			// fmt.Println("Col 1", string(tempCols[1]))
			// fmt.Println("Col 2", string(tempCols[2]))
			// fmt.Println("Col 3", string(tempCols[3]))
			// fmt.Printf(
			// 	"Row %d; Col %d; ColLen: %d; Char: %s; MaxLen: %d; prevChar: %s; seenTwoSpace: %t; charCounts: %+v\n\n",
			// 	i,
			// 	currentCol,
			// 	len(tempCols[currentCol]),
			// 	string(char),
			// 	getColMaxLength(currentCol),
			// 	string(prevChar),
			// 	seenTwoSpace,
			// 	charCounts,
			// )

			prevChar = byte(char)
		}

		cols[0] = append(cols[0], tempCols[0]...)
		cols[1] = append(cols[1], tempCols[1]...)
		cols[2] = append(cols[2], tempCols[2]...)
		cols[3] = append(cols[3], tempCols[3]...)

		cols[0] = append(cols[0], '\n')
		cols[1] = append(cols[1], '\n')
		cols[2] = append(cols[2], '\n')
		cols[3] = append(cols[3], '\n')
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

func HandleSimpleCases() {

	// doc.Case = trimExp.ReplaceAllString(string(cols[1]), "")
	// doc.Nature = trimExp.ReplaceAllString(string(cols[2]), "")
	// doc.Accord = trimExp.ReplaceAllString(string(cols[3]), "")
	/*cols := spaceExp.Split(str, -1)

	if i == 0 && len(cols) < 4 {
		return handleColsWithOneSpace(data, &doc)
	}

	if i == 0 {
		doc.Case = cols[1]
		doc.Nature = cols[2]
		doc.Accord = cols[3]
		continue
	}

	if len(cols) > 2 {
		doc.Nature = doc.Nature + "\n" + cols[1]
		doc.Nature = doc.Nature + "\n" + cols[2]
		continue
	}

	doc.Accord = doc.Accord + "\n" + cols[1]
	*/
}
