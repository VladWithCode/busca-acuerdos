package tsj

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	db "github.com/vladwithcode/juzgados/internal/db/docs"
	"github.com/vladwithcode/juzgados/internal/reader"
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
	spaceExp := regexp.MustCompile(" {2,}")
	rows := lineExp.Split(string(data), -1)

	doc := db.Doc{}

	for i, str := range rows {
		cols := spaceExp.Split(str, -1)

		if i == 0 {
			doc.ID = uuid.New().String()
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
	}

	return &doc
}
