package manager

import (
	"fmt"
	"time"
	"github.com/google/uuid"
	"github.com/LarsFox/motovskikh-hse-backend/entities"
)

type db interface {
	GetTest(testID string) (*entities.Test, error)
	SaveAttempt(attempt *entities.Attempt) error
	GetTestStats(testID string) (*entities.TestStats, error)
	GetUserPercentile(testID string, percentage float64, timeSpent int) (float64, error)
	GetRecentAttempts(testID, userHash string, period time.Duration) (int, error)
	
	GetQuestions(testID string) ([]entities.Question, error)
	GetQuestion(questionID string) (*entities.Question, error)
	CheckAnswer(questionID, userAnswer string) (bool, int, error)
	SaveUserAnswers(answers []*entities.UserAnswer) error
	GetUserAnswers(attemptID string) ([]entities.UserAnswer, error)
	GetDetailedAnalysis(attemptID string) (*entities.DetailedAnalysis, error)
  GetTimePercentile(testID string, timeSpent int) (float64, error)
	CreateTestData() error
}

type Manager struct {
	db db
}

func New(db db) *Manager {
	return &Manager{db: db}
}

// GetTest возвращает тест.
func (m *Manager) GetTest(testID string) (*entities.Test, error) {
	return m.db.GetTest(testID)
}

// GetTestAnalysis возвращает анализ.
func (m *Manager) GetTestAnalysis(testID string, userPercentage float64, userTimeSpent int) (*entities.TestStats, float64, error) {
	stats, err := m.db.GetTestStats(testID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get test stats: %w", err)
	}

	percentile, err := m.db.GetUserPercentile(testID, userPercentage, userTimeSpent)
	if err != nil {
		return stats, 0, fmt.Errorf("failed to get user percentile: %w", err)
	}
	
	return stats, percentile, nil
}

// CreateTestData создает тестовые данные.
func (m *Manager) CreateTestData() error {
	return m.db.CreateTestData()
}

// Возвращает недавние попытки.
func (m *Manager) GetRecentAttempts(testID, userHash string, period time.Duration) (int, error) {
	return m.db.GetRecentAttempts(testID, userHash, period)
}

// ValidateAttempt проверяет валидность попытки.
func (m *Manager) ValidateAttempt(testID, userHash string, percentage float64, timeSpent int) (bool, string) {
	// Проверка минимального времени (60 секунд).
	if timeSpent < 60 {
		return false, "Time spent is too short (minimum 60 seconds)"
	}
	
	// Проверка минимального процента (5%).
	if percentage < 5 {
		return false, "Score is too low (minimum 5%)"
	}
	
	// Проверка на спам (не более 3 попыток в час).
	recentAttempts, err := m.GetRecentAttempts(testID, userHash, time.Hour)
	if err == nil && recentAttempts >= 3 {
		return false, "Too many attempts recently (maximum 3 per hour)"
	}

	return true, ""
}

// Отправка теста.
func (m *Manager) SubmitTestWithValidation(testID, userHash string, answers map[string]interface{}, timeSpent int) (string, map[string]interface{}, error) {
	// Получаем вопросы теста.
	questions, err := m.db.GetQuestions(testID)
	if err != nil {
		return "", nil, err
	}
	
	// Проверяем ответы.
	totalScore := 0
	maxScore := 0
	var userAnswers []*entities.UserAnswer
	attemptID := uuid.New().String()
	
	for _, question := range questions {
		maxScore += question.Points
		
		if userAnswer, ok := answers[question.ID]; ok {
			answerStr := fmt.Sprintf("%v", userAnswer)
			isCorrect, points, _ := m.db.CheckAnswer(question.ID, answerStr)
			
			if isCorrect {
				totalScore += points
			}
			
			userAnswers = append(userAnswers, &entities.UserAnswer{
				ID:          uuid.New().String(),
				AttemptID:   attemptID,
				QuestionID:  question.ID,
				UserAnswer:  answerStr,
				IsCorrect:   isCorrect,
				PointsEarned: points,
				CreatedAt:   time.Now(),
			})
		}
	}
	
	// Рассчитываем процент.
	percentage := 0.0
	if maxScore > 0 {
		percentage = float64(totalScore) / float64(maxScore) * 100
	}
	
	// Проверяем, можно ли сохранять эту попытку.
	isValid, validationMessage := m.ValidateAttempt(testID, userHash, percentage, timeSpent)
	
	// Если попытка невалидна.
	if !isValid {
		return "", nil, fmt.Errorf("attempt validation failed: %s", validationMessage)
	}
	
	// Сохраняем попытку.
	attempt := &entities.Attempt{
		ID:         attemptID,
		TestID:     testID,
		UserHash:   userHash,
		Score:      totalScore,
		MaxScore:   maxScore,
		Percentage: percentage,
		TimeSpent:  timeSpent,
		Answers:    "{}",
		IsValid:    true,
		CreatedAt:  time.Now(),
	}
	
	err = m.db.SaveAttempt(attempt)
	if err != nil {
		return "", nil, err
	}
	
	// Сохраняем детальные ответы.
	if len(userAnswers) > 0 {
		m.db.SaveUserAnswers(userAnswers)
	}
	
	// Готовим результат.
	result := map[string]interface{}{
		"attempt_id":         attemptID,
		"score":              totalScore,
		"max_score":          maxScore,
		"percentage":         percentage,
		"questions_total":    len(questions),
		"questions_answered": len(userAnswers),
		"is_valid":           true,
		"validation_message": "Попытка успешно проверена и сохранена",
	}
	// Добавляем быстрый анализ.
	quickAnalysis := m.generateQuickAnalysis(attemptID, testID, percentage, timeSpent, userHash)
	result["quick_analysis"] = quickAnalysis
	
	return attemptID, result, nil
}

func (m *Manager) SubmitTestResult(testName, userHash string, percentage float64, timeSpent int) (string, map[string]interface{}, error) {
	score := int(percentage)
	maxScore := 100
	
	// Проверяем валидность попытки.
	isValid, validationMessage := m.ValidateAttempt(testName, userHash, percentage, timeSpent)
	if !isValid {
		return "", nil, fmt.Errorf("attempt validation failed: %s", validationMessage)
	}
	
	// Генерируем ID попытки.
	attemptID := uuid.New().String()
	
	// Сохраняем попытку.
	attempt := &entities.Attempt{
		ID:         attemptID,
		TestID:     testName,
		UserHash:   userHash,
		Score:      score,
		MaxScore:   maxScore,
		Percentage: percentage,
		TimeSpent:  timeSpent,
		Answers:    "{}",
		IsValid:    true,
		CreatedAt:  time.Now(),
	}
	
	err := m.db.SaveAttempt(attempt)
	if err != nil {
		return "", nil, err
	}

	// Готовим результат.
	result := map[string]interface{}{
			"score":              score,
			"max_score":          maxScore,
			"percentage":         percentage,
			"time_spent":         timeSpent,
			"is_valid":           true,
			"validation_message": "Попытка успешно сохранена",
			"quick_analysis": m.generateQuickAnalysis(attemptID, testName, percentage, timeSpent, userHash),
	}

	return attemptID, result, nil
}

// GetDetailedAnalysis возвращает детальный анализ.
func (m *Manager) GetDetailedAnalysis(attemptID string) (*entities.DetailedAnalysis, error) {
    return m.db.GetDetailedAnalysis(attemptID)
}

// GetQuestions возвращает вопросы теста.
func (m *Manager) GetQuestions(testID string) ([]entities.Question, error) {
	return m.db.GetQuestions(testID)
}

// generateQuickAnalysis создает быстрый анализ для ответа на submit.
func (m *Manager) generateQuickAnalysis(attemptID, testID string, percentage float64, timeSpent int, userHash string) map[string]interface{} {
    stats, percentile, err := m.GetTestAnalysis(testID, percentage, timeSpent)
    if err != nil {
        // Возвращаем базовый анализ при ошибке.
        return map[string]interface{}{
            "percentile":  0,
            "category":    "unknown",
            "better_than": 0,
            "message":     "Анализ временно недоступен",
        }
    }
    
    category := "average"
    if percentile >= 90 {
        category = "excellent"
    } else if percentile >= 70 {
        category = "good"
    } else if percentile < 30 {
        category = "needs_improvement"
    }
    
    return map[string]interface{}{
        "percentile":          percentile,
        "category":            category,
        "better_than":         int(percentile),
        "average_percentage":  stats.AvgPercentage,
        "average_time":        stats.AvgTimeSpent,
        "message":             m.getCategoryMessage(category, percentile),
    }
}

func (m *Manager) getCategoryMessage(category string, percentile float64) string {
	messages := map[string]string{
		"excellent":        fmt.Sprintf("Превосходно! Вы лучше, чем %.0f%% участников!", percentile),
		"good":             fmt.Sprintf("Хороший результат! Вы лучше %.0f%% участников.", percentile),
		"average":          fmt.Sprintf("Средний результат. Вы в середине распределения."),
		"needs_improvement": fmt.Sprintf("Есть над чем поработать. Вы лучше %.0f%% участников.", percentile),
	}
	
	if msg, ok := messages[category]; ok {
		return msg
	}
	return "Спасибо за прохождение теста!"
}