package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"time"
)

var (
	helpFlag bool
	timeout  int64 //耗时
	size     int   //大小
	count    int   //次数
	typ      uint8 = 8
	code     uint8 = 0
	SendCnt  int
	RecCnt   int
	MaxTime  int64 = math.MinInt64 //最大耗时
	MinTime  int64 = math.MaxInt64
	SumTime  int64
)

// icmp
type ICMP struct {
	Type        uint8  //类型
	Code        uint8  //代码
	CheckSum    uint16 //校验和
	ID          uint16 //id
	SequenceNum uint16 //序号
}

func main() {
	fmt.Println()
	log.SetFlags(log.Llongfile)
	//fmt.Println(os.Args)
	GetCommendFlags()

	//打印帮助信息
	if helpFlag {
		displayHelp()
		os.Exit(0)
	}

	//获取目标ip
	desIP := os.Args[len(os.Args)-1]
	conn, err := net.DialTimeout("ip:icmp", desIP, time.Duration(timeout)*time.Millisecond)
	//conn, err := net.DialTimeout("ip:icmp", desIP, time.Duration(timeout))
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer conn.Close()

	//远程地址
	remoteaddr := conn.RemoteAddr()
	fmt.Printf("正在ping %s [%s] 具有 %d 字节的数据:\n", desIP, remoteaddr, size)
	for i := 0; i < count; i++ {
		icmp := &ICMP{
			Type:        typ,
			Code:        code,
			CheckSum:    uint16(0),
			ID:          uint16(i),
			SequenceNum: uint16(i),
		}
		//将请求数据转为二进制流
		var buffer bytes.Buffer
		binary.Write(&buffer, binary.BigEndian, icmp)
		//请求数据
		data := make([]byte, size)
		//将请求数据写到icmp报文后面
		buffer.Write(data)
		data = buffer.Bytes()
		// ICMP 请求签名（校验和）：相邻两位拼接到一起，拼接成两个字节的数
		checkSum := checkSum(data)
		//签名赋值到data
		data[2] = byte(checkSum >> 8)
		data[3] = byte(checkSum)
		startTime := time.Now()

		//设超时时间
		conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond))

		//将data写入连接中
		_, err := conn.Write(data)
		if err != nil {
			log.Panicln(err)
			continue
		}
		SendCnt++

		//接受响应
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		//fmt.Println(buf)
		//fmt.Println(n)
		if err != nil {
			log.Println(err)
			continue
		}
		RecCnt++
		//打印
		t := time.Since(startTime).Milliseconds()
		fmt.Printf("来自 %d.%d.%d.%d 的回复：字节=%d 时间=%d TTL=%d\n", buf[12], buf[13], buf[14], buf[15], n-28, t, buf[8])
		MaxTime = Max(MaxTime, t)
		MinTime = Min(MinTime, t)
		SumTime += t
		time.Sleep(time.Second)
	}
	fmt.Printf("\n %s 的 Ping 统计信息:", remoteaddr)
	fmt.Printf("    数据包: 已发送 = %d，已接收 = %d，丢失 = %d (%.f%% 丢失)，\n",
		SendCnt, RecCnt, count*2-SendCnt-RecCnt, float64(count*2-SendCnt-RecCnt)/float64(count*2)*100)
	fmt.Printf("往返行程的估计时间(以毫秒为单位):\n")
	fmt.Printf("    最短 = %d，最长 = 3%d，平均 = %d", MinTime, MaxTime, SumTime/int64(count))
}
func GetCommendFlags() {
	flag.Int64Var(&timeout, "w", 1000, "请求超时时间")
	flag.IntVar(&size, "l", 32, "发送字节数")
	flag.IntVar(&count, "n", 4, "请求次数")
	flag.BoolVar(&helpFlag, "h", false, "显示帮助信息")
	flag.Parse()
}

func displayHelp() {
	fmt.Println(`选项：
	-n count                     要发送的回显请求数。
	-l size                      发送的缓冲区大小。
	-w timeout					 等待每次回复的超时时间(毫秒)。				
	-h 	                         帮助选项
	`)
}

func checkSum(data []byte) uint16 {
	//第一步两两拼接并求和
	length := len(data)
	index := 0
	var sum uint32
	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		length -= 2
		index += 2
	}
	//奇数情况，还剩一个，直接求和过去
	if length == 1 {
		sum += uint32(data[index])
	}
	//第二步：高 16 位，低 16 位 相加，直至高 16 位为 0
	hi := sum >> 16
	for hi != 0 {
		sum = hi + uint32(uint16(sum))
		hi = sum >> 16
	}
	// 返回 sum 值 取反
	return uint16(^sum)
}
func Max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func Min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
