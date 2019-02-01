package utils

import (
	"encoding/json"
	"fmt"
	"github.com/irisnet/irishub-sync/logger"
	"math"
	"strconv"
)

func ParseInt(text string) (i int64, b bool) {
	i, err := strconv.ParseInt(text, 10, 0)
	if err != nil {
		return i, false
	}
	return i, true
}

func ParseUint(text string) (i int64, b bool) {
	i, ok := ParseInt(text)
	if ok {
		return i, i > 0
	}
	return i, ok
}

func RoundFloat(num float64, bit int) (i float64, b bool) {
	format := "%" + fmt.Sprintf("0.%d", bit) + "f"
	s := fmt.Sprintf(format, num)
	i, err := strconv.ParseFloat(s, 0)
	if err != nil {
		return i, false
	}
	return i, true
}

func Round(x float64) int64 {
	return int64(math.Floor(x + 0.5))
}

func Map2Struct(srcMap map[string]interface{}, obj interface{}) {
	bz, err := json.Marshal(srcMap)
	if err != nil {
		logger.Error("map convert to struct failed")
	}
	err = json.Unmarshal(bz, obj)
	if err != nil {
		logger.Error("map convert to struct failed")
	}
}
