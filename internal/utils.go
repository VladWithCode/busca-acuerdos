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
		mStr string = fmt.Sprint(m)
		dStr string = fmt.Sprint(d)
	)

	if m < 10 {
		mStr = MONTHS[m-1]
	}

	if d < 10 {
		dStr = fmt.Sprintf("0%d", d)
	}

	return fmt.Sprintf("%v de %v de %v", dStr, mStr, y)
}
