package bot

import (
	"crypto/md5"
	"io"
	"strings"

	"github.com/gofrs/uuid/v5"
)

func UniqueObjectId(args ...string) string {
	h := md5.New()
	for _, s := range args {
		io.WriteString(h, s)
	}
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	id, err := uuid.FromBytes(sum)
	if err != nil {
		panic(err)
	}
	return id.String()
}

func UniqueConversationId(userId, recipientId string) string {
	minId, maxId := userId, recipientId
	if strings.Compare(userId, recipientId) > 0 {
		maxId, minId = userId, recipientId
	}
	h := md5.New()
	io.WriteString(h, minId)
	io.WriteString(h, maxId)
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	id, _ := UuidFromBytes(sum)
	return id.String()
}

func Chunked(source []interface{}, size int) [][]interface{} {
	var result [][]interface{}
	index := 0
	for index < len(source) {
		end := index + size
		if end >= len(source) {
			end = len(source)
		}
		result = append(result, source[index:end])
		index += size
	}
	return result
}
