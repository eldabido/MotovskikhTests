package api

import (
	"encoding/json"
	"net/http"
	"fmt"
	"github.com/LarsFox/motovskikh-hse-backend/entities"
	"math"
	"strings"
)

type GetAnalysisRequest struct {
	TestID     string  `json:"test_id"`
	Percentage float64 `json:"percentage"`
	TimeSpent  int     `json:"time_spent"`
}

// МЕТОДЫ-ХЭНДЛЕРЫ.

// hndlrGetAnalysis возвращает анализ результатов.
func (m *Manager) hndlrGetAnalysis(w http.ResponseWriter, r *http.Request) {
	var req GetAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	
	// Валидация.
	if req.TestID == "" || req.Percentage < 0 || req.Percentage > 100 || req.TimeSpent <= 0 {
		m.sendError(w, http.StatusBadRequest, "Invalid parameters")
		return
	}
	
	// Получаем анализ.
	stats, percentile, err := m.manager.GetTestAnalysis(req.TestID, req.Percentage, req.TimeSpent)
	if err != nil {
		m.sendError(w, http.StatusInternalServerError, "Failed to get analysis")
		return
	}
	
	response := map[string]interface{}{
		"your_percentage": req.Percentage,
		"your_time":       req.TimeSpent,
		"percentile":      percentile,
		"better_than":     int(percentile),
	}
	
	if stats != nil {
		response["average_percentage"] = stats.AvgPercentage
		response["average_time"] = stats.AvgTimeSpent
		response["total_attempts"] = stats.TotalAttempts
	}
	
	m.send(w, response)
}

// hndlrGetAdvancedAnalysis возвращает расширенный анализ.
func (m *Manager) hndlrGetAdvancedAnalysis(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AttemptID string `json:"attempt_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	
	if req.AttemptID == "" {
		m.sendError(w, http.StatusBadRequest, "attempt_id is required")
		return
	}
	
	// Получаем детальный анализ из менеджера.
	analysis, err := m.manager.GetDetailedAnalysis(req.AttemptID)
	if err != nil {
		m.sendError(w, http.StatusInternalServerError, "Failed to get analysis: "+err.Error())
		return
	}
	
	// Форматируем.
	response := map[string]interface{}{
		"analysis": analysis,
		"summary": m.formatAnalysisSummary(analysis),
		"comparison": m.formatComparisonData(analysis),
	}
	
	m.send(w, response)
}

// formatAnalysisSummary форматирует анализ.
func (m *Manager) formatAnalysisSummary(analysis *entities.DetailedAnalysis) map[string]interface{} {
	return map[string]interface{}{
		"skill_level": analysis.SkillLevel,
		"category":    analysis.DistributionCategory.Name,
		"message":     m.generateSummaryMessage(analysis),
	}
}

// formatComparisonData форматирует данные для сравнения.
func (m *Manager) formatComparisonData(analysis *entities.DetailedAnalysis) map[string]interface{} {
	return map[string]interface{}{
		"percentile_rank":   fmt.Sprintf("%.1f%%", analysis.PercentileRank),
		"time_percentile":   fmt.Sprintf("%.1f%%", analysis.TimePercentile),
		"better_than_users": fmt.Sprintf("%.0f%%", analysis.PercentileRank),
		"faster_than_users": fmt.Sprintf("%.0f%%", analysis.TimePercentile),
		"vs_average": map[string]interface{}{
			"score_diff":     analysis.Percentage - analysis.TestStats.AvgPercentage,
			"time_diff":      float64(analysis.TimeSpent) - analysis.TestStats.AvgTimeSpent,
			"score_status":   m.getDiffStatus(analysis.Percentage, analysis.TestStats.AvgPercentage),
			"time_status":    m.getTimeDiffStatus(float64(analysis.TimeSpent), analysis.TestStats.AvgTimeSpent),
		},
	}
}

// generateSummaryMessage генерирует сообщение для пользователя.
func (m *Manager) generateSummaryMessage(analysis *entities.DetailedAnalysis) string {
	var messages []string
	
	// Основное сообщение по категории.
	switch analysis.DistributionCategory.Name {
	case "elite":
		messages = append(messages, "Это элита. Превосходный результат!")
	case "excellent":
		messages = append(messages, "Отличный результат! Вы в топ-10% участников.")
	case "good":
		messages = append(messages, "Хорошая работа! Вы лучше большинства участников.")
	case "average":
		messages = append(messages, "Средний результат. Есть, куда расти.")
	case "below_average":
		messages = append(messages, "Ниже среднего. Побольше практики.")
	case "needs_improvement":
		messages = append(messages, "Плохо.")
	}
	
	// Добавляем сравнение со средним.
	avgDiff := analysis.Percentage - analysis.TestStats.AvgPercentage
	if avgDiff > 10 {
		messages = append(messages, fmt.Sprintf("На %.1f%% выше среднего показателя!", avgDiff))
	} else if avgDiff < -10 {
		messages = append(messages, fmt.Sprintf("На %.1f%% ниже среднего показателя.", math.Abs(avgDiff)))
	}
	
	return strings.Join(messages, " ")
}

// getDiffStatus определяет статус разницы.
func (m *Manager) getDiffStatus(userValue, avgValue float64) string {
	diff := userValue - avgValue
	if diff > 10 {
		return "significantly_higher"
	} else if diff > 5 {
		return "higher"
	} else if diff < -10 {
		return "significantly_lower"
	} else if diff < -5 {
		return "lower"
	}
	return "similar"
}

// getTimeDiffStatus определяет статус разницы во времени.
func (m *Manager) getTimeDiffStatus(userTime, avgTime float64) string {
	diff := userTime - avgTime
	if diff < -30 {
		return "much_faster"
	} else if diff < -10 {
		return "faster"
	} else if diff > 30 {
		return "much_slower"
	} else if diff > 10 {
		return "slower"
	}
	return "similar"
}

// hndlrCreateTestData создает тестовые данные.
func (m *Manager) hndlrCreateTestData(w http.ResponseWriter, r *http.Request) {
	if err := m.manager.CreateTestData(); err != nil {
		m.sendError(w, http.StatusInternalServerError, "Failed to create test data: "+err.Error())
		return
	}
	
	// Получаем список всех созданных тестов для отображения.
	testIDs := []string{"1"}
	
	// Проверяем существование каждого теста.
	availableTests := []map[string]interface{}{}
	for _, testID := range testIDs {
		test, err := m.manager.GetTest(testID)
		if err == nil && test != nil {
			availableTests = append(availableTests, map[string]interface{}{
				"id":   test.ID,
				"name": test.Name,
				"url":  "/tests/get/?test_id=" + testID,
			})
		}
	}
	
	m.send(w, map[string]interface{}{
		"test_data_created": true,
		"message": "Тестовые данные созданы",
		"available_tests": availableTests,
		"example_requests": map[string]string{
			"get_test": "GET /tests/get/?test_id=1",
			"submit_test": "POST /tests/submit/ with JSON body",
		},
	})
}