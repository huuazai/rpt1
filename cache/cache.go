package cache

import (
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const (
	B = 1 << (iota * 10)
	KB
	MB
	GB
	TB
	PB
)

type cache struct {
	//最大可使用内存
	maxMemorySize int64
	//最大可使用内存字符串表示
	maxMemorySizeString string
	//当前已经使用的内存
	currMemorysize int64
	//锁
	locker sync.RWMutex
	//key value 键值对数据
	values map[string]*cacheval
}

// value
type cacheval struct {
	//数据值
	val any
	//过期时间
	expiretime time.Time
	//大小
	size int64
}

// 实例化
func NewCache() *cache {
	return &cache{
		values: make(map[string]*cacheval, 100),
	}
}

// 设置最大可用内存
func (ch *cache) SetMaxMemory(size string) bool {
	ch.maxMemorySize, ch.maxMemorySizeString = parseSize(size)
	return true
}
func parseSize(size string) (int64, string) {
	re, _ := regexp.Compile("[0-9]+")
	unit := string(re.ReplaceAll([]byte(size), []byte("")))
	num, _ := strconv.ParseInt(strings.Replace(size, unit, "", 1), 10, 64)
	unit = strings.ToUpper(unit)
	var bytenum int64
	switch unit {
	case "B":
		bytenum = num
	case "KB":
		bytenum = num * KB
	case "MB":
		bytenum = num * MB
	case "GB":
		bytenum = num * GB
	case "TB":
		bytenum = num * TB
	case "PB":
		bytenum = num * PB
	default:
		num = 0
	}
	if num == 0 {
		log.Println("内存设置错误！！！")
		num = 100
		unit = "MB"
		bytenum = num * MB
	}
	bytestr := strconv.FormatInt(num, 10) + unit
	return bytenum, bytestr
}

// 添加或修改key
func (ch *cache) Set(key string, value any, expire time.Duration) bool {
	ch.locker.Lock()
	defer ch.locker.Unlock()
	v := &cacheval{
		val:        value,
		expiretime: time.Now().Add(expire),
		size:       getSize(value),
	}
	ch.Del(key)
	ch.values[key] = v
	ch.currMemorysize += v.size
	return true
}
func getSize(value any) int64 {
	return unsafe.Sizeof(value)
}

// 获取key的value值
func (ch *cache) Get(key string) (*cacheval, bool) {

	ch.locker.RLock()
	defer ch.locker.RUnlock()
	ok := ch.Exist(key)
	if ok && time.Now().Before(ch.values[key].expiretime) {
		return ch.values[key], true
	}
	return &cacheval{}, false
}

// 删除key
func (ch *cache) Del(key string) bool {
	val, ok := ch.Get(key)
	if ok {
		ch.locker.Lock()
		defer ch.locker.Unlock()
		ch.currMemorysize -= val.size
		delete(ch.values, key)

	}
	return true
}

// 判断key是否存在
func (ch *cache) Exist(key string) bool {
	_, ok := ch.values[key]
	if ok {
		return true
	}
	return false
}

// 清空所有的key
func (ch *cache) Flush() bool {
	ch.locker.Lock()
	defer ch.locker.Unlock()
	ch.values = nil
}

// 获取所有key的数量
func (ch *cache) keys() int {
	return len(ch.values)
}
