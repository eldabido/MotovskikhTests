package api

import (
	"encoding/json"
	"net/http"
	"github.com/google/uuid"
)

type SubmitTestRequest struct {
	TestName    string  `json:"test_name"`
	Percentage  float64 `json:"percentage"`
	TimeSpent   int     `json:"time_spent"`
}

// МЕТОДЫ-ХЭНДЛЕРЫ.

// hndlrGetTest возвращает информацию о тесте для прохождения.
func (m *Manager) hndlrGetTest(w http.ResponseWriter, r *http.Request) {
	testName := r.URL.Query().Get("name")
	language := r.URL.Query().Get("language")
	
	if testName == "" {
		testName = "europe"
	}
	if language == "" {
		language = "ru"
	}
	
	response := m.getMockedTestResponse(testName, language)
	
	m.send(w, response)
}

// getMockedTestResponse возвращает захардкоженный тест.
func (m *Manager) getMockedTestResponse(testName, language string) map[string]interface{} {
	if testName == "europe" {
		return map[string]interface{}{
			"test": map[string]interface{}{
				"id":   "europe",
				"name": "europe",
				"type": "geography",
				"i18n": map[string]interface{}{
					"ru": map[string]interface{}{
						"title": "Европа",
						"desc":  "Тест на знание стран Европы",
					},
					"en": map[string]interface{}{
						"title": "Europe",
						"desc":  "European countries test",
					},
				},
				"settings": map[string]interface{}{
					"modes":        []string{"flags", "capitals", "regions"},
					"width":        1529,
					"height":       843,
					"international": true,
					"workshop":     false,
					"raskraska":    false,
				},
			},
			"questions": []map[string]interface{}{
				{
					"id":            "france_flag",
					"question_type": "flag",
					"question_text": "Чей это флаг?",
					"options":       []string{"Франция", "Германия", "Италия", "Испания"},
					"points":        1,
					"metadata": map[string]interface{}{
						"region": "Западная Европа",
					},
				},
				{
					"id":            "germany_capital",
					"question_type": "capital",
					"question_text": "Столица Германии?",
					"options":       []string{"Берлин", "Париж", "Мадрид", "Рим"},
					"points":        1,
					"metadata": map[string]interface{}{
						"region": "Центральная Европа",
					},
				},
			},
			"total_questions": 2,
			"instructions": map[string]string{
				"ru": "Определите страну по флагу или назовите столицу",
				"en": "Identify the country by flag or name the capital",
			},
		}
	}
	
	// Дефолтный ответ.
	return map[string]interface{}{
		"test": map[string]interface{}{
			"id":   testName,
			"name": testName,
			"type": "general",
			"i18n": map[string]interface{}{
				language: map[string]interface{}{
					"title": testName,
					"desc":  "Общий тест",
				},
			},
			"settings": map[string]interface{}{
				"modes": []string{"general"},
			},
		},
		"questions": []map[string]interface{}{},
		"total_questions": 0,
		"instructions": map[string]string{
			"ru": "Ответьте на вопросы теста",
			"en": "Answer the test questions",
		},
	}
}


// hndlrSubmitTest - хэндлер отправки теста.
func (m *Manager) hndlrSubmitTest(w http.ResponseWriter, r *http.Request) {
	var req SubmitTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	
	// Валидация.
	if req.TestName == "" || req.TimeSpent <= 0 || req.Percentage < 0 || req.Percentage > 100 {
		m.sendError(w, http.StatusBadRequest, "Missing or invalid required fields: test_name, time_spent > 0, percentage 0-100")
		return
	}
	
	if req.TimeSpent < 30 {
		m.sendError(w, http.StatusBadRequest, "Time spent is too short (minimum 30 seconds)")
		return
	}
	
	// Получаем или генерируем userHash.
	userHash := r.Header.Get("X-User-Hash")
	if userHash == "" {
		userHash = "anonymous_" + uuid.New().String()[:8]
	}
	
	// Передаем процент и время.
	attemptID, result, err := m.manager.SubmitTestResult(req.TestName, userHash, req.Percentage, req.TimeSpent)
	if err != nil {
		m.sendError(w, http.StatusInternalServerError, "Failed to submit test: "+err.Error())
		return
	}

	result["attempt_id"] = attemptID
	m.send(w, result)
}