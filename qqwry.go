package qqwry

import (
	"io/ioutil"
	"fmt"
	"strings"
	"strconv"
	"bytes"
	"github.com/yinheli/mahonia"
	"sort"
)

type QQwry struct {
	Data []byte
	IndexBegin int
	IndexEnd int
	IndexCount int
	Idx1 []int
	Idx2 []int
	Idxo []int
}

func int3(data []byte, offset int) int {
	return int(data[offset]) + int(data[offset+1]) << 8 + int(data[offset+2]) << 16
}

func int4(data []byte, offset int) int {
	return int(data[offset]) + int(data[offset+1]) << 8 + int(data[offset+2]) << 16 + int(data[offset+3]) << 24
}

func inetAton(ip string) int64 {
	bits := strings.Split(ip, ".")

	b0, _ := strconv.Atoi(bits[0])
	b1, _ := strconv.Atoi(bits[1])
	b2, _ := strconv.Atoi(bits[2])
	b3, _ := strconv.Atoi(bits[3])

	var sum int64

	sum += int64(b0) << 24
	sum += int64(b1) << 16
	sum += int64(b2) << 8
	sum += int64(b3)

	return sum
}

func NewQQwry(file string) (q *QQwry) {
	q = &QQwry{}
	var err error
	q.Data, err = ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	if len(q.Data) < 8 {
		panic("file size is too small")
	}

	q.IndexBegin = int4(q.Data, 0)
	q.IndexEnd = int4(q.Data, 4)
	if q.IndexBegin > q.IndexEnd || (q.IndexEnd - q.IndexBegin) % 7 != 0 || q.IndexEnd + 7 > len(q.Data) {
		panic("index error")
	}

	q.IndexCount = (q.IndexEnd - q.IndexBegin) / 7 + 1

	for i:=0;i<q.IndexCount;i++ {
		ipBegin := int4(q.Data, q.IndexBegin + i * 7)
		offset := int3(q.Data, q.IndexBegin + i * 7 + 4)
		ipEnd := int4(q.Data, offset)
		q.Idx1 = append(q.Idx1, ipBegin)
		q.Idx2 = append(q.Idx2, ipEnd)
		q.Idxo = append(q.Idxo, offset + 4)
	}

	fmt.Printf("%s %d bytes, %d segments. with index.", file, len(q.Data), len(q.Idx1))

	return q
}

func (q *QQwry) Find(ip string) (country string, province string) {
	return q.indexSearch(inetAton(ip))
}

//func (qqwry *QQwry)rawFind(ip int64) (country string, province string) {
//	l := 0
//	r := qqwry.IndexCount
//	for {
//		if r - l <= 1 {
//			break
//		}
//		m := (l + r) / 2
//		offset := qqwry.IndexBegin + m * 7
//		newIp := int4(qqwry.Data, offset)
//		if ip < int64(newIp) {
//			r = m
//		} else {
//			l = m
//		}
//	}
//	offset := qqwry.IndexBegin + 7 * l
//	ipBegin := int4(qqwry.Data, offset)
//	offset = int3(qqwry.Data, offset+4)
//	ipEnd := int4(qqwry.Data, offset)
//
//	if ip >= int64(ipBegin) && ip <= int64(ipEnd) {
//		return qqwry.getAddr(offset+4)
//	} else {
//		return "", ""
//	}
//}

func (q *QQwry)indexSearch(ip int64) (country string, province string) {
	posi := sort.Search(len(q.Idx1), func(i int) bool { return int64(q.Idx1[i]) > ip }) - 1
	if posi >= 0 && ip >= int64(q.Idx1[posi]) && ip <= int64(q.Idx2[posi]) {
		return q.getAddr(q.Idxo[posi])
	} else {
		return "", ""
	}
}

func (q *QQwry)getAddr(offset int) (country string, province string) {
	// mode 0x01, full jump
	mode := q.Data[offset]
	if mode == 1 {
		offset = int3(q.Data, offset+1)
		mode = q.Data[offset]
	}

	var c []byte
	// country
	if mode == 2 {
		off1 := int3(q.Data, offset+1)
		c = q.Data[off1:(bytes.IndexByte(q.Data[off1:], 0) + off1)]
		offset += 4
	} else {
		c = q.Data[offset:(bytes.IndexByte(q.Data[offset:], 0) + offset)]
		offset += len(c) + 1
	}

	// province
	if q.Data[offset] == 2 {
		offset = int3(q.Data, offset+1)
	}
	p := q.Data[offset:(bytes.IndexByte(q.Data[offset:], 0) + offset)]

	gbk := mahonia.NewDecoder("gbk")
	country = gbk.ConvertString(string(c))
	province = gbk.ConvertString(string(p))
	utf8 := mahonia.NewDecoder("utf-8")
	_, c, _ = utf8.Translate([]byte(country), true)
	_, p, _ = utf8.Translate([]byte(province), true)
	return string(c), string(p)
}
