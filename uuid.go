package bot

import (
	"log"

	"github.com/satori/go.uuid"
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

func UuidFromBytes(input []byte) string {
	return uuid.FromBytesOrNil(input).String()
}
