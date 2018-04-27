package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
)

type Error struct {
	Status      int    `json:"status"`
	Code        int    `json:"code"`
	Description string `json:"description"`
	trace       string
}

func (sessionError Error) Error() string {
	str, err := json.Marshal(sessionError)
	if err != nil {
		log.Panicln(err)
	}
	return string(str)
}

func BlazeServerError(ctx context.Context, err error) Error {
	description := "Blaze server error."
	return createError(ctx, http.StatusInternalServerError, 7000, description, err)
}

func createError(ctx context.Context, status, code int, description string, err error) Error {
	pc, file, line, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()
	trace := fmt.Sprintf("[ERROR %d] %s\n%s:%d", code, description, file, line)
	if err != nil {
		if sessionError, ok := err.(Error); ok {
			trace = trace + "\n" + sessionError.trace
		} else {
			trace = trace + "\n" + err.Error()
		}
	}
	print(funcName)

	return Error{
		Status:      status,
		Code:        code,
		Description: description,
		trace:       trace,
	}
}
