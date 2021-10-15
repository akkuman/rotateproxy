package rotateproxy

import (
	"math/rand"
	"sync"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func RandomSyncMap(sMap sync.Map) (key, value interface{}) {
	var tmp [][2]interface{}
	sMap.Range(func(key, value interface{}) bool {
		if value.(int) == 0 {
			tmp = append(tmp, [2]interface{}{key, value})
		}
		return true
	})
	element := tmp[rand.Intn(len(tmp))]
	return element[0], element[1]
}

func SyncMapIsBlank(sMap sync.Map) bool {
	isBlank := true
	sMap.Range(func(key, value interface{}) bool {
		if value.(int) == 0 {
			isBlank = false
			return false
		}
		return true
	})
	return isBlank
}
