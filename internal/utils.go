package internal

import (
	"fmt"
	"time"
)

var MONTHS = [12]string{
	"Enero",
	"Febrero",
	"Marzo",
	"Abril",
	"Mayo",
	"Junio",
	"Julio",
	"Agosto",
	"Septiembre",
	"Octubre",
	"Noviembre",
	"Diciembre",
}

func FormatDate(date time.Time) string {
	var (
		d    int    = date.Day()
		m    int    = int(date.Month())
		y    int    = date.Year()
		mStr string = MONTHS[m-1]
	)

	return fmt.Sprintf("%02d de %v de %d", d, mStr, y)
}

func FormatTimestampToString(ts time.Time) string {
	dateStr := FormatDate(ts)

	var (
		h int = ts.Hour()
		m int = ts.Minute()
		s int = ts.Second()
	)

	return fmt.Sprintf("%s %02d:%02d:%02d", dateStr, h, m, s)
}
