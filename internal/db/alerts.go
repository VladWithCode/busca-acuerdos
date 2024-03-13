package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Alert struct {
	Id             string         `json:"id" db:"id"`
	UserId         string         `json:"userId" db:"user_id"`
	CaseId         string         `json:"caseId" db:"case_id"`
	NatureCode     string         `json:"natureCode" db:"nature_code"`
	Active         bool           `json:"active" db:"active"`
	Alias          sql.NullString `json:"alias" db:"alias"`
	LastUpdatedAt  time.Time      `json:"lastUpdateAt" db:"last_updated_at"`
	LastCheckedAt  time.Time      `json:"lastCheckedAt" db:"last_checked_at"`
	LastAccord     sql.NullString `json:"lastAccord" db:"last_accord"`
	LastAccordDate sql.NullTime   `json:"lastAccordDate" db:"last_accord_date"`

	/**
	TODO: Future improvements
	frequency: daily | ... | monthly
	subscribers
	...
	*/
}

// type AutoReportAlerts map[string][]Alert
type AutoReportAlert struct {
	Id             string         `json:"id" db:"id"`
	CaseId         string         `json:"caseId" db:"case_id"`
	NatureCode     string         `json:"natureCode" db:"nature_code"`
	LastAccord     sql.NullString `json:"lastAccord" db:"last_accord"`
	LastAccordDate sql.NullTime   `json:"lastAccordDate" db:"last_accord_date"`
}

type AutoReportUser struct {
	Id       string
	Name     string
	Lastname string
	Email    string
	Phone    string
	Alerts   []AutoReportAlert
}

func FindAlertById(id string) (*Alert, error) {
	conn, err := GetPool()
	defer conn.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err != nil {
		return nil, err
	}

	var alert Alert

	row, err := conn.Query(
		ctx,
		"SELECT * FROM alerts WHERE id = $1 LIMIT 1",
		id,
	)

	if err != nil {
		return nil, err
	}

	alert, err = pgx.CollectOneRow[Alert](row, pgx.RowToStructByName[Alert])

	if err != nil {
		return nil, err
	}

	return &alert, nil
}

func FindAlertsByUser(userId string) (*[]Alert, error) {
	conn, err := GetPool()
	defer conn.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err != nil {
		return nil, err
	}

	var alerts []Alert

	rows, err := conn.Query(
		ctx,
		"SELECT * FROM alerts WHERE user_id = $1",
		userId,
	)

	if err != nil {
		return nil, err
	}

	// Refer to https://stackoverflow.com/questions/61704842/how-to-scan-a-queryrow-into-a-struct-with-pgx
	alerts, err = pgx.CollectRows[Alert](rows, pgx.RowToStructByName[Alert])

	if err != nil {
		return nil, err
	}

	return &alerts, nil
}

func FindAutoReportAlertsForUser(userId string) (*[]Alert, error) {
	conn, err := GetPool()
	defer conn.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err != nil {
		return nil, err
	}

	var alerts []Alert

	rows, err := conn.Query(
		ctx,
		"SELECT * FROM alerts WHERE user_id = $1 AND active = TRUE",
		userId,
	)

	if err != nil {
		return nil, err
	}

	// Refer to https://stackoverflow.com/questions/61704842/how-to-scan-a-queryrow-into-a-struct-with-pgx
	alerts, err = pgx.CollectRows[Alert](rows, pgx.RowToStructByName[Alert])

	if err != nil {
		return nil, err
	}

	return &alerts, nil
}

func FindAutoReportAlertsWithUserData() ([]*AutoReportUser, error) {
	conn, err := GetPool()
	defer conn.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err != nil {
		return nil, err
	}

	var resultUsers = []*AutoReportUser{}

	rows, err := conn.Query(ctx, "SELECT users.id, users.name, users.lastname, users.email, users.phone_number, ARRAY_AGG((alerts.id, alerts.case_id, alerts.nature_code, alerts.last_accord, alerts.last_accord_date)) AS alerts FROM users LEFT JOIN alerts ON users.id = alerts.user_id WHERE alerts.active = true GROUP BY users.id, users.id, users.name, users.lastname, users.email, users.phone_number;")

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var user AutoReportUser
		var tempArr pgtype.Array[AutoReportAlert]

		err = rows.Scan(&user.Id, &user.Name, &user.Lastname, &user.Email, &user.Phone, &tempArr)

		if err != nil {
			return nil, err
		}

		user.Alerts = tempArr.Elements

		resultUsers = append(resultUsers, &user)
	}

	return resultUsers, nil
}

func CreateAlert(userId string, caseId string, natureCode string) (*Alert, error) {
	conn, err := GetPool()
	defer conn.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err != nil {
		return nil, err
	}

	var alert Alert
	id, err := uuid.NewV7()

	if err != nil {
		return nil, err
	}

	row, err := conn.Query(
		ctx,
		"INSERT INTO alerts (id, user_id, case_id, nature_code, active) VALUES ($1, $2, $3, $4, $5) RETURNING *",
		id,
		userId,
		caseId,
		natureCode,
		true,
	)

	if err != nil {
		return nil, err
	}

	alert, err = pgx.CollectExactlyOneRow[Alert](row, pgx.RowToStructByName[Alert])

	if err != nil {
		return nil, err
	}

	return &alert, nil
}

func UpdateAlertAccord(userId, caseId, natureCode string, updatedAlert *Alert) error {
	conn, err := GetPool()
	defer conn.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err != nil {
		return err
	}

	res, err := conn.Exec(
		ctx,
		"UPDATE alerts SET last_accord = $1, last_updated_at = $2, last_checked_at = $3 WHERE user_id = $4 AND case_id = $5 AND nature_code = $6",
		updatedAlert.LastAccord,
		updatedAlert.LastUpdatedAt,
		updatedAlert.LastCheckedAt,
		userId,
		caseId,
		natureCode,
	)

	if err != nil {
		return err
	}

	rowsAffected := res.RowsAffected()

	if rowsAffected == 0 {
		return errors.New("No se encontró la alerta solicitada")
	}

	return nil
}
