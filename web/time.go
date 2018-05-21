package web

import (
	"math"
	"strconv"
	"time"
)

func TimeElapsed(timestamp time.Time, full bool) string {
	var precise [8]string // this an array, not slice
	var text string
	var future bool // our crystal ball

	// get years, months and days
	// and get hours, minutes, seconds
	now := time.Now()
	year2, month2, day2 := now.Date()
	hour2, minute2, second2 := now.Clock()

	year1, month1, day1 := timestamp.Date()
	hour1, minute1, second1 := timestamp.Clock()

	// are we forecasting the future?
	if (year2 - year1) < 0 {
		future = true
	}

	if (month2 - month1) < 0 {
		future = true
	}
	if (day2 - day1) < 0 {
		future = true
	}
	if (hour2 - hour1) < 0 {
		future = true
	}
	if (minute2 - minute1) < 0 {
		future = true
	}
	if (second2 - second1) < 0 {
		future = true
	}

	// convert negative to positive numbers
	year := math.Abs(float64(int(year2 - year1)))
	month := math.Abs(float64(int(month2 - month1)))
	day := math.Abs(float64(int(day2 - day1)))
	hour := math.Abs(float64(int(hour2 - hour1)))
	minute := math.Abs(float64(int(minute2 - minute1)))
	second := math.Abs(float64(int(second2 - second1)))

	week := math.Floor(day / 7)

	// Ouch!, no if-else short hand - see https://golang.org/doc/faq#Does_Go_have_a_ternary_form

	if year > 0 {
		if int(year) == 1 {
			precise[0] = strconv.Itoa(int(year)) + " year"
		} else {
			precise[0] = strconv.Itoa(int(year)) + " years"
		}
	}

	if month > 0 {
		if int(month) == 1 {
			precise[1] = strconv.Itoa(int(month)) + " month"
		} else {
			precise[1] = strconv.Itoa(int(month)) + " months"
		}
	}

	if week > 0 {
		if int(week) == 1 {
			precise[2] = strconv.Itoa(int(week)) + " week"
		} else {
			precise[2] = strconv.Itoa(int(week)) + " weeks"
		}
	}

	if day > 0 {
		if int(day) == 1 {
			precise[3] = strconv.Itoa(int(day)) + " day"
		} else {
			precise[3] = strconv.Itoa(int(day)) + " days"
		}
	}

	if hour > 0 {
		if int(hour) == 1 {
			precise[4] = strconv.Itoa(int(hour)) + " hour"
		} else {
			precise[4] = strconv.Itoa(int(hour)) + " hours"
		}
	}

	if minute > 0 {
		if int(minute) == 1 {
			precise[5] = strconv.Itoa(int(minute)) + " minute"
		} else {
			precise[5] = strconv.Itoa(int(minute)) + " minutes"
		}
	}

	if second > 0 {
		if int(second) == 1 {
			precise[6] = strconv.Itoa(int(second)) + " second"
		} else {
			precise[6] = strconv.Itoa(int(second)) + " seconds"
		}
	}

	for _, v := range precise {
		if v != "" {
			// no comma after second
			if v[len(v)-5:len(v)-1] != "cond" {
				precise[7] += v + ", "
			} else {
				precise[7] += v
			}
		}
	}

	if !future {
		text = " ago"
	} else {
		return "a few minutes ago"
	}

	if full {
		return precise[7] + text
	} else {
		// return the first non-empty position
		for k, v := range precise {
			if v != "" {
				return precise[k] + text
			}
		}
	}
	return "invalid date"
}
