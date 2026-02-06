package bot

import (
	"crypto/md5"
	"io"
	"slices"
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

func GroupConversationId(ownerId, groupName string, participants []string, randomId string) string {
	randomId = uuid.Must(uuid.FromString(randomId)).String()
	gid := UniqueConversationId(ownerId, groupName)
	gid = UniqueConversationId(gid, randomId)

	slices.Sort(participants)
	for _, p := range participants {
		gid = UniqueConversationId(gid, p)
	}
	return gid
}

func Chunked(source []any, size int) [][]any {
	var result [][]any
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

func MakeUniqueStringSlice(ss []string) []string {
	unique := make([]string, 0)
	filter := make(map[string]bool)
	for _, s := range ss {
		if filter[s] {
			continue
		}
		unique = append(unique, s)
		filter[s] = true
	}
	return unique
}
