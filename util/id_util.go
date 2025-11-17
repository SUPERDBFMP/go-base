package util

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

func GenerateBigintID() int64 {
	currentTime := time.Now().Format("20060102150405.000")
	timePart := currentTime[:14] + currentTime[15:] // 移除小数点,得到17位时间
	randomPart := rand.Intn(100)                    // 2位随机数
	id, _ := strconv.ParseInt(fmt.Sprintf("%s%02d", timePart, randomPart), 10, 64)
	return id
}
