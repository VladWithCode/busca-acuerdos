package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type Alert struct {
	Id             string         `json:"id" db:"id"`
	UserId         string         `json:"userId" db:"user_id"`
	CaseId         string         `json:"caseId" db:"case_id"`
	Nature         string         `json:"nature" db:"nature"`
	NatureCode     string         `json:"natureCode" db:"nature_code"`
	Active         bool           `json:"active" db:"active"`
	Alias          sql.NullString `json:"alias" db:"alias"`
	LastUpdatedAt  time.Time      `json:"lastUpdateAt" db:"last_updated_at"`
	LastCheckedAt  time.Time      `json:"lastCheckedAt" db:"last_checked_at"`
	LastAccord     sql.NullString `json:"lastAccord" db:"last_accord"`
	LastAccordDate sql.NullTime   `json:"lastAccordDate" db:"last_accord_date"`
	CreatedAt      time.Time      `json:"createdAt" db:"created_at"`

	/**
	TODO: Future improvements
	frequency: daily | ... | monthly
	subscribers
	...
	*/
}

func (a *Alert) GetCaseKey() string {
	return a.CaseId + "+" + a.NatureCode
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
	if err != nil {
		return nil, err
	}

	defer conn.Release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

func FindAlertsByUser(userId string, findActive bool) ([]*Alert, error) {
	conn, err := GetPool()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var rows pgx.Rows

	if findActive {
		rows, err = conn.Query(
			ctx,
			"SELECT * FROM alerts WHERE user_id = $1 AND active = TRUE",
			userId,
		)
	} else {
		rows, err = conn.Query(
			ctx,
			"SELECT * FROM alerts WHERE user_id = $1",
			userId,
		)
	}

	if err != nil {
		return nil, err
	}

	// Refer to https://stackoverflow.com/questions/61704842/how-to-scan-a-queryrow-into-a-struct-with-pgx
	alerts, err := pgx.CollectRows[Alert](rows, pgx.RowToStructByName[Alert])

	if err != nil {
		return nil, err
	}

	resAlerts := []*Alert{}

	for _, al := range alerts {
		newAl := al
		resAlerts = append(resAlerts, &newAl)
	}

	return resAlerts, nil
}

func FindAutoReportAlertsForUser(userId string) (*[]Alert, error) {
	conn, err := GetPool()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

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
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var resultUsers = []*AutoReportUser{}

	rows, err := conn.Query(ctx, "SELECT users.id, users.name, users.lastname, users.email, users.phone_number, ARRAY_AGG((alerts.id, alerts.case_id, alerts.nature_code, alerts.last_accord, alerts.last_accord_date)) AS alerts FROM users LEFT JOIN alerts ON users.id = alerts.user_id WHERE alerts.active = true AND users.phone_number IS NOT NULL GROUP BY users.id, users.id, users.name, users.lastname, users.email, users.phone_number;")

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

func CreateAlertWithData(data *Alert) (*Alert, error) {
	conn, err := GetPool()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	id, err := uuid.NewV7()

	if err != nil {
		return nil, err
	}

	t, err := conn.Exec(
		ctx,
		"INSERT INTO alerts (id, user_id, case_id, nature_code, active, last_accord, last_accord_date, alias, nature) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		id,
		data.UserId,
		data.CaseId,
		data.NatureCode,
		data.Active,
		data.LastAccord,
		data.LastAccordDate,
		data.Alias,
		data.Nature,
	)

	if err != nil {
		return nil, err
	}

	if t.RowsAffected() == 0 {
		return nil, errors.New("No se creó la alerta")
	}

	data.Id = id.String()
	return data, nil
}

func CreateAlert(userId string, caseId string, natureCode string) (*Alert, error) {
	conn, err := GetPool()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

func UpdateAlertAccords(alertsData []*Alert) error {
	conn, err := GetPool()
	if err != nil {
		return err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	queryBatch := pgx.Batch{}

	for _, alert := range alertsData {
		queryBatch.Queue(
			"UPDATE alerts SET last_updated_at = NOW(), last_checked_at = NOW(), last_accord = $1, last_accord_date = $2, nature = $3 WHERE user_id = $4 AND case_id = $5 AND nature_code = $6",
			alert.LastAccord.String,
			alert.LastAccordDate.Time,
			alert.Nature,
			alert.UserId,
			alert.CaseId,
			alert.NatureCode,
		).Exec(func(ct pgconn.CommandTag) error {
			if ct.RowsAffected() == 0 {
				cK := alert.CaseId + "+" + alert.NatureCode
				return errors.New(fmt.Sprintf("No se pudo actualizar alerta para el caso %v", cK))
			}

			return nil
		})
	}

	err = conn.SendBatch(ctx, &queryBatch).Close()

	if err != nil {
		return err
	}

	return nil
}

func UpdateAlertAccord(userId, caseId, natureCode string, updatedAlert *Alert) error {
	conn, err := GetPool()
	if err != nil {
		return err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := conn.Exec(
		ctx,
		"UPDATE alerts SET last_accord = $1, last_updated_at = $2, last_checked_at = $3, nature = $4 WHERE user_id = $5 AND case_id = $6 AND nature_code = $7",
		updatedAlert.LastAccord,
		updatedAlert.LastUpdatedAt,
		updatedAlert.LastCheckedAt,
		updatedAlert.Nature,
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

func DeleteAlertById(id string) error {
	conn, err := GetPool()
	if err != nil {
		return err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := conn.Exec(
		ctx,
		"DELETE FROM alerts WHERE id = $1",
		id,
	)

	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return errors.New("No se encontro alerta con el id especificado")
	}

	return nil
}

func DeleteUserAlertById(id, userId string) error {
	conn, err := GetPool()
	if err != nil {
		return err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := conn.Exec(
		ctx,
		"DELETE FROM alerts WHERE id = $1 AND user_id = $2",
		id,
		userId,
	)

	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return errors.New("No se encontro alerta con el id especificado")
	}

	return nil
}
