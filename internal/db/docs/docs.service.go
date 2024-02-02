package db

import (
	"context"
	"fmt"
	"time"

	"github.com/vladwithcode/juzgados/internal/db"
)

func FetchDocForCase(caseID string) {
}

func GetDocs() ([]Doc, error) {
	conn, err := db.GetPool()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(ctx, "SELECT * FROM docs")
	docs := []Doc{}

	if err != nil {
		fmt.Println("Query Err")
		return nil, err
	}

	for rows.Next() {
		doc := Doc{}

		if err := rows.Scan(
			&doc.ID,
			&doc.Case,
			&doc.Nature,
			&doc.NatureCode,
			&doc.Accord,
			&doc.AccordDate,
		); err != nil {
			fmt.Println("Iter Err")
			return nil, err
		}

		docs = append(docs, doc)
	}

	return docs, err
}

func GetDocByID(id string) (*Doc, error) {
	conn, err := db.GetPool()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err != nil {
		return nil, err
	}

	doc := Doc{}

	row := conn.QueryRow(ctx, "SELECT * FROM docs WHERE id = $1", id)

	err = row.Scan(&doc.ID, &doc.Nature, &doc.NatureCode, &doc.Case, &doc.Accord, &doc.AccordDate)

	if err != nil {
		return nil, err
	}

	return &doc, nil
}

func GetDocByCase(caseID string) (*Doc, error) {
	conn, err := db.GetPool()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err != nil {
		return nil, err
	}

	doc := Doc{}

	row := conn.QueryRow(ctx, "SELECT * FROM docs WHERE case_id = $1", caseID)

	err = row.Scan(&doc.ID, &doc.Nature, &doc.NatureCode, &doc.Case, &doc.Accord, &doc.AccordDate)

	if err != nil {
		return nil, err
	}

	return &doc, nil
}

func CreateDoc(id, case_id, nature, natureCode, accord string, date time.Time) error {
	conn, err := db.GetPool()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err != nil {
		return err
	}

	_, err = conn.Exec(
		ctx,
		"INSERT INTO docs (id, case_id, nature, nature_code, accord, accord_date) VALUES ($1, $2, $3, $4, $5, $6)",
		id,
		case_id,
		nature,
		natureCode,
		accord,
		date,
	)

	if err != nil {
		return err
	}

	return nil
}
