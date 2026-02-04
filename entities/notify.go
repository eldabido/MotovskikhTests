package entities

import (
	"errors"
	"log"
)

var (
	ErrNotFound     = errors.New("something is not found")
	ErrInvalidInput = errors.New("your input is invalid")
)

// Notify сообщает об ошибке.
func Notify(e error, meta ...map[string]any) {
	if len(meta) == 0 {
		log.Println(e)
		return
	}

	for _, extras := range meta {
		log.Println(e, extras)
	}
}
