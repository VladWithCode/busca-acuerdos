package tsj

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/vladwithcode/juzgados/internal/db"
	"github.com/vladwithcode/juzgados/internal/reader"
)

const (
	IDX_LEN    = 7
	CASE_LEN   = 15
	NATURE_LEN = 23
	ACCORD_LEN = 100
)

const DEFAULT_DAYS_BACK = 62
const EXTENDED_DAYS_BACK = 93

type NotFoundError struct {
	Msg string
}

func (e *NotFoundError) Error() string {
	return e.Msg
}

// For optimal doc search, this struct holds information about the cases pending search
// as well as the Docs generated for the cases found.
// This way, multiple case searches can be done on the same file fetched from TSJ
// avoiding multiple net requests for the same file
// PendingCases maps in the form of [caseType]: [caseKey1, caseKey2, ..., caseKeyN]
// e.g. [mer2]: [12/2003, 45/2006]
type MultiCaseSearch struct {
	PendingCases map[string][]string
	Docs         []*db.Doc

	mux sync.Mutex
}

type GetCasesResult struct {
	Docs         []*db.Doc
	NotFoundKeys []string
	mux          sync.Mutex
}

func (r *GetCasesResult) AppendCase(caseDoc *db.Doc) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.Docs = append(r.Docs, caseDoc)
}

func (r *GetCasesResult) AppendNotFound(key string) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.NotFoundKeys = append(r.NotFoundKeys, key)
}

func GenRegExp(caseId string) (*regexp.Regexp, error) {
	return regexp.Compile(fmt.Sprintf(`(?m)^(\d+.+%v[^\n]+(?:[\n][^\d].*)+)`, caseId))
}

func GetCaseData(caseId, caseType string, searchDate *time.Time, daysBack int) (*db.Doc, error) {
	var localDate time.Time

	if searchDate == nil {
		localDate = time.Now()
	} else {
		localDate = *searchDate
	}

	var data []byte
	var err error

	for i := 0; i <= daysBack; i++ {
		y, m, d := localDate.Date()
		date := fmt.Sprintf("%d%d%d", d, m, y)
		data, err = FetchAndReadDoc(caseId, date, caseType)

		if data != nil {
			break
		}

		localDate = localDate.AddDate(0, 0, -1)

		// For every iteration except the last, reset err
		if i < daysBack {
			err = nil
		}
	}

	if err != nil {
		return nil, err
	}

	doc := DataToDoc(data)

	doc.Case = caseId
	doc.AccordDate = localDate
	doc.NatureCode = caseType

	return doc, nil
}

func GetCasesDataV1(caseKeys []string, daysBack uint, startDate time.Time) (*GetCasesResult, error) {
	result := GetCasesResult{
		Docs:         []*db.Doc{},
		NotFoundKeys: []string{},
		mux:          sync.Mutex{},
	}
	wg := sync.WaitGroup{}

	for _, cK := range caseKeys {
		wg.Add(1)
		go func(cK string) {
			defer wg.Done()
			params := strings.Split(cK, "+")
			caseId, caseType := params[0], params[1]

			doc, err := GetCaseData(caseId, caseType, &startDate, int(daysBack))

			if err != nil {
				result.AppendNotFound(cK)
				return
			}

			result.AppendCase(doc)
		}(cK)
	}

	wg.Wait()

	return &result, nil
}

// This approach attempts to improve efficency
// by reducing the net request for TSJ reports
// By only fetching the file for a specified
// date and caseType once and then executing
// all the searches for the pending case Ids
// GetCasesDataV2
func GetCasesData(caseKeys []string, daysBack uint, startDate time.Time) (*GetCasesResult, error) {
	searchData := MultiCaseSearch{
		PendingCases: genCaseMap(caseKeys),
	}
	wg := sync.WaitGroup{}

	for cType, cIds := range searchData.PendingCases {
		wg.Add(1)
		go func(cType string, cIds []string, startDate time.Time, daysBack uint) {
			defer wg.Done()
			var pendingIds []string
			iDaysBack := int(daysBack + 1)

			for i := 0; i <= iDaysBack; i++ {
				tsjFile, err := reader.Reader(startDate.Format("212006"), cType)

				// When an error is found, skip to next try
				if err != nil {
					// Only print errors for the last try
					if i == iDaysBack {
						fmt.Printf("[%v on date %v] Failed to find file %v\n", cType, startDate.Format("02/01/06"), err)
					}
					continue
				}

				if tsjFile != nil {
					for _, cId := range cIds {
						searchExp, _ := GenRegExp(cId)
						idxs := searchExp.FindIndex(*tsjFile)

						if idxs == nil {
							pendingIds = append(pendingIds, cId)
							continue
						}

						start, end := idxs[0], idxs[1]

						doc := DataToDoc((*tsjFile)[start:end])
						doc.NatureCode = cType

						searchData.mux.Lock()
						searchData.Docs = append(searchData.Docs, doc)
						searchData.mux.Unlock()
					}
				}

				if len(pendingIds) == 0 {
					break
				}

				cIds = pendingIds
				startDate = startDate.AddDate(0, 0, -1)
			}
		}(cType, cIds, startDate, daysBack)
	}

	wg.Wait()

	return &GetCasesResult{
		Docs: searchData.Docs,
	}, nil
}

func FetchAndReadDoc(caseId, searchDate, caseType string) ([]byte, error) {
	pdfContent, err := reader.Reader(searchDate, caseType)

	if err != nil {
		return nil, err
	}

	searchExp, err := GenRegExp(caseId)

	if err != nil {
		return nil, err
	}

	idx := searchExp.FindIndex(*pdfContent)

	if len(idx) == 0 {
		err := &NotFoundError{
			Msg: "No se encontró información sobre el caso solicitado",
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
	pageExp := regexp.MustCompile(`PAGINA :[^\n]+/`)
	rows := lineExp.Split(string(data), -1)

	doc := db.Doc{}
	doc.ID = uuid.New().String()

	var cols [4][]byte
	var tempCols [4][]byte
	var currentCol int
	var seenTwoSpace bool
	var prevChar byte
	charCounts := []int{0, 0, 0, ACCORD_LEN}

	for rowidx, str := range rows {
		if len(str) == 0 {
			break
		}

		if pageExp.Match([]byte(str)) {
			break
		}

		currentCol = 0
		seenTwoSpace = false
		prevChar = 0
		tempCols[0] = []byte{}
		tempCols[1] = []byte{}
		tempCols[2] = []byte{}
		tempCols[3] = []byte{}

		for charIdx, char := range str {
			if seenTwoSpace && char != ' ' && currentCol < 3 {
				currentCol++
				seenTwoSpace = false
				prevChar = 0
			}

			tempCols[currentCol] = utf8.AppendRune(tempCols[currentCol], char)

			// Keep track of the length of the columns for splitting following rows
			if rowidx == 0 && currentCol < 3 {
				charCounts[currentCol]++
			}

			if char == ' ' && prevChar == ' ' {
				seenTwoSpace = true
			}

			colLen := len(tempCols[currentCol])
			maxLen := getColMaxLength(currentCol)

			if char == ' ' && str[ensureSafeIndex(charIdx+1, len(str))] != ' ' && colLen >= maxLen && currentCol < 3 {
				currentCol++
				seenTwoSpace = false
				prevChar = 0 // byte(0) == ''
				continue
			}

			// Check if the length of the current column is at the max length
			if rowidx > 0 && colLen == charCounts[currentCol] {
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
		cols[3] = append(cols[3], byte(' '))
	}

	doc.Case = strings.TrimSpace(strings.TrimLeft(string(cols[1]), "0"))
	doc.Nature = strings.TrimSpace(string(cols[2]))
	doc.Accord = strings.TrimSpace(string(cols[3]))

	return &doc
}

func genCaseMap(caseKeys []string) map[string][]string {
	caseMap := map[string][]string{}

	for _, cK := range caseKeys {
		params := strings.Split(cK, "+")
		cId, cType := params[0], params[1]

		if _, ok := caseMap[cType]; !ok {
			caseMap[cType] = []string{}
		}

		caseMap[cType] = append(caseMap[cType], cId)
	}

	return caseMap
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
