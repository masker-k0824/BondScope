package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseJapaneseDate は "R6.4.11" や "H20.1.5" を time.Time に変換します
func ParseJapaneseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// 元号(1文字)とそれ以外に分ける
	era := s[0:1]
	body := s[1:]
	parts := strings.Split(body, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", s)
	}

	year, errY := strconv.Atoi(parts[0])
	month, errM := strconv.Atoi(parts[1])
	day, errD := strconv.Atoi(parts[2])

	if errY != nil || errM != nil || errD != nil {
		return time.Time{}, fmt.Errorf("failed to parse date numbers: %s", s)
	}

	var westernYear int
	switch era {
	case "R": // 令和 (2019年が1年)
		westernYear = year + 2018
	case "H": // 平成 (1989年が1年)
		westernYear = year + 1988
	case "S": // 昭和 (1926年が1年)
		westernYear = year + 1925
	default:
		return time.Time{}, fmt.Errorf("unknown era: %s", era)
	}

	// time.Date を直接使って生成（これが最も安全）
	// 時刻は 00:00:00、場所はローカル（またはUTC）に設定
	return time.Date(westernYear, time.Month(month), day, 0, 0, 0, 0, time.Local), nil
}
