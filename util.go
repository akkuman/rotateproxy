package rotateproxy

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"
)

var errInvalidWrite = errors.New("invalid write result")

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

func IsProxyURLBlank() bool {
	proxies, err := QueryAvailProxyURL()
	if err != nil {
		fmt.Printf("[!] Error: %v\n", err)
		return false
	}
	return len(proxies) == 0
}

// copyBuffer is the actual implementation of Copy and CopyBuffer.
// if buf is nil, one is allocated.
func CopyBufferWithCloseErr(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	if buf != nil && len(buf) == 0 {
		panic("empty buffer in CopyBuffer")
	}
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}
	if buf == nil {
		size := 32 * 1024
		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			// 正常关闭时也返回错误
			// if er != io.EOF {
			// 	err = er
			// }
			err = er
			break
		}
	}
	return written, err
}