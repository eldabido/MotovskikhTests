package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"runtime/debug"
	"strings"
	"fmt"
	"github.com/LarsFox/motovskikh-hse-backend/entities"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

const (
	// Бэйс-64 накладывает оверхед примерно треть от файла, 33.33%
	// https://stackoverflow.com/questions/34109053/what-file-size-is-data-if-its-450kb-base64-encoded
	// Максимальный размер устанавливаю 1 МБ * 1.4, размер кодирования бэйс-64 с запасом.
	// В Нжинксе указываю такое же значение.
	maxBodySize = 1468006
)

// Внутренние ошибки.
var (
	errUnknownError = errors.New("unknown error")
)

func notify(e error, meta ...map[string]any) {
	if strings.Contains(e.Error(), "write: broken pipe") {
		return
	}

	if errors.Is(e, context.Canceled) {
		return
	}

	entities.Notify(e, meta...)
}

func notifyRecover(meta ...map[string]any) {
	rec := recover()
	if rec == nil {
		return
	}

	var err error
	switch t := rec.(type) {
	case error:
		err = t
	default:
		err = errUnknownError
	}

	meta = append(meta, map[string]any{
		"stack": string(debug.Stack()),
	})

	notify(err, meta...)
}

// unmarshalParams парсит входящие в методы API параметры и валидирует их.
func unmarshalParams(r *http.Request, prms runtime.Validatable) error {
	if err := json.NewDecoder(r.Body).Decode(prms); err != nil {
		return err
	}

	if err := r.Body.Close(); err != nil {
		return err
	}

	if err := prms.Validate(strfmt.Default); err != nil {
		return err
	}

	return nil
}

func (m *Manager) sendError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":    false,
		"error": message,
	})
}

// sendErrorPage возвращает страницу ошибки.
func (m *Manager) sendErrorPage(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	w.Write([]byte(fmt.Sprintf("nope, %d", code)))
}