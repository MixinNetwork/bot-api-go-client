package bot

import (
	"crypto/md5"
	"io"
	"log"

	"github.com/gofrs/uuid/v5"
)

var Nil = uuid.Nil

func UuidNewV4() uuid.UUID {
	id, err := uuid.NewV4()
	if err != nil {
		log.Panicln(err)
	}
	return id
}

func UuidFromString(id string) (uuid.UUID, error) {
	return uuid.FromString(id)
}

func UuidFromBytes(input []byte) (uuid.UUID, error) {
	return uuid.FromBytes(input)
}

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
