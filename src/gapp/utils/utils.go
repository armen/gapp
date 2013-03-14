package utils

import (
	humanize "github.com/dustin/go-humanize"

	"crypto/rand"
	"fmt"
	"io"
	"time"
)

func GenId(length uint) string {

	buf := make([]byte, length)
	io.ReadFull(rand.Reader, buf)

	return fmt.Sprintf("%x", buf)
}

func HumanizeTime(then int64) string {
	return humanize.Time(time.Unix(then, 0))
}
