package mysql

import (
	"fmt"
	"strings"
	"time"
	"log"
	"math/rand"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"github.com/LarsFox/motovskikh-hse-backend/entities"
	"github.com/google/uuid"
)

type Config struct {
	Host    string `envconfig:"optional"`
	Pass    string
	MaxConn int `envconfig:"default=0"`
	Name    string
	User    string
}

type Client struct {
	db *gorm.DB
}

func (c *Config) connection() string {
	sqlHost := c.Host
	if !strings.Contains(sqlHost, "tcp") {
		sqlHost = fmt.Sprintf("tcp(%s)", sqlHost)
	}
	return fmt.Sprintf("%s:%s@%s/%s?charset=utf8mb4&parseTime=True&loc=Local", c.User, c.Pass, sqlHost, c.Name)
}

func NewClient(cfg *Config) (*Client, error) {
    db, err := gorm.Open(mysql.Open(cfg.connection()))
    if err != nil {
      return nil, fmt.Errorf("dbs new client err: %w", err)
    }

    // Автомиграция для всех таблиц.
    err = db.AutoMigrate(
        &entities.Attempt{},
        &entities.TestStats{},
    )
    if err != nil {
      return nil, fmt.Errorf("dbs migrate err: %w", err)
    }

    d, err := db.DB()
    if err != nil {
      return nil, fmt.Errorf("dbs new client err: %w", err)
    }

    d.SetMaxOpenConns(cfg.MaxConn)
    return &Client{db: db}, nil
}



// GetTest возвращает информацию о тесте из таблицы tests.
func (c *Client) GetTest(testID string) (*entities.Test, error) {
    var test entities.Test
    err := c.db.Table("tests").Where("name = ?", testID).First(&test).Error
    if err == nil {
      return &test, nil
    }
    
    var id int
    _, err = fmt.Sscanf(testID, "%d", &id)
    if err != nil {
      return nil, fmt.Errorf("test not found by name")
    }
    return &test, nil
}

// SaveAttempt сохраняет попытку.
func (c *Client) SaveAttempt(attempt *entities.Attempt) error {
	return c.db.Create(attempt).Error
}

// GetTestStats - получает статистику теста.
func (c *Client) GetTestStats(testID string) (*entities.TestStats, error) {
	var stats entities.TestStats
	
	// Ищем статистику.
	err := c.db.Where("test_id = ?", testID).First(&stats).Error
	
	if err == gorm.ErrRecordNotFound {
		// Если статистики нет, создаем пустую.
		log.Printf("No stats found for test %s, creating empty...", testID)
		
		stats = entities.TestStats{
			ID:            uuid.New().String(),
			TestID:        testID,
			Date:          time.Now(),
			TotalAttempts: 0,
			ValidAttempts: 0,
			AvgPercentage: 0,
			AvgTimeSpent:  0,
		}
		
		// Сохраняем пустую статистику.
		if err := c.db.Create(&stats).Error; err != nil {
			log.Printf("Failed to create empty stats: %v", err)
		}
		
		return &stats, nil
		
	} else if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	
	return &stats, nil
}

// GetUserPercentile - вычисляет перцентиль пользователя.
func (c *Client) GetUserPercentile(testID string, percentage float64, timeSpent int) (float64, error) {
	// Сначала проверяем есть ли данные.
	var count int64
	err := c.db.Model(&entities.Attempt{}).
		Where("test_id = ? AND is_valid = ?", testID, true).
		Count(&count).Error
	
	if err != nil {
		return 0, fmt.Errorf("failed to count attempts: %w", err)
	}
	
	if count == 0 {
		// Нет данных.
		return 50.0, nil
	}
	
	// Считаем сколько попыток хуже.
	var worseCount int64
	err = c.db.Model(&entities.Attempt{}).
		Where("test_id = ? AND is_valid = ?", testID, true).
		Where("percentage < ? OR (percentage = ? AND time_spent > ?)", 
			percentage, percentage, timeSpent).
		Count(&worseCount).Error
	
	if err != nil {
		return 0, fmt.Errorf("failed to count worse attempts: %w", err)
	}
	
	// Вычисляем перцентиль.
	percentile := (float64(worseCount) / float64(count)) * 100.0
	
	// Добавляем 0.5 для корректного округления.
	percentile += 0.5
	
	if percentile > 100 {
		percentile = 100
	}
	
	log.Printf("Percentile for test %s: %.1f%% (worse: %d, total: %d)", 
		testID, percentile, worseCount, count)
	
	return percentile, nil
}

// GetRecentAttempts возвращает количество попыток пользователя за последний период.
func (c *Client) GetRecentAttempts(testID, userHash string, period time.Duration) (int, error) {
	var count int64
	timeLimit := time.Now().Add(-period)
	
	err := c.db.Model(&entities.Attempt{}).
		Where("test_id = ? AND user_hash = ? AND created_at > ?", 
			testID, userHash, timeLimit).
		Count(&count).Error
	
	return int(count), err
}

// GetQuestions возвращает вопросы для теста.
func (c *Client) GetQuestions(testID string) ([]entities.Question, error) {
	var questions []entities.Question
	
	var id int
	_, err := fmt.Sscanf(testID, "%d", &id)
	
	if err == nil {
		err = c.db.Where("test_id = ?", id).Find(&questions).Error
	} else {
		err = c.db.Where("test_id = ?", testID).Find(&questions).Error
	}
	
	return questions, err
}

// GetQuestion возвращает конкретный вопрос.
func (c *Client) GetQuestion(questionID string) (*entities.Question, error) {
	var question entities.Question
	err := c.db.Where("id = ?", questionID).First(&question).Error
	return &question, err
}

// CheckAnswer проверяет правильность ответа на вопрос.
func (c *Client) CheckAnswer(questionID, userAnswer string) (bool, int, error) {
	question, err := c.GetQuestion(questionID)
	if err != nil {
		return false, 0, err
	}
	
	// Проверка.
	isCorrect := strings.EqualFold(strings.TrimSpace(userAnswer), strings.TrimSpace(question.CorrectAnswer))
	
	points := 0
	if isCorrect {
		points = question.Points
	}
	
	return isCorrect, points, nil
}

// SaveUserAnswers сохраняет детальные ответы пользователя.
func (c *Client) SaveUserAnswers(answers []*entities.UserAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	return c.db.CreateInBatches(answers, 100).Error
}

// GetUserAnswers возвращает ответы пользователя для попытки.
func (c *Client) GetUserAnswers(attemptID string) ([]entities.UserAnswer, error) {
	var answers []entities.UserAnswer
	err := c.db.Where("attempt_id = ?", attemptID).Find(&answers).Error
	return answers, err
}

// GetDetailedAnalysis возвращает полный детальный анализ попытки.
func (c *Client) GetDetailedAnalysis(attemptID string) (*entities.DetailedAnalysis, error) {
	// Получаем попытку.
	var attempt entities.Attempt
	err := c.db.Where("id = ?", attemptID).First(&attempt).Error
	if err != nil {
		return nil, err
	}
	
	// Получаем детальные ответы.
	userAnswers, err := c.GetUserAnswers(attemptID)
	if err != nil {
		return nil, err
	}
	
	// Получаем статистику теста.
	testStats, err := c.GetTestStats(attempt.TestID)
	if err != nil {
		testStats = &entities.TestStats{}
	}
	
	// Получаем перцентили.
	percentileRank, err := c.GetUserPercentile(attempt.TestID, attempt.Percentage, attempt.TimeSpent)
	if err != nil {
		percentileRank = 0
	}
	
	// Получаем перцентиль по времени.
	timePercentile, err := c.GetTimePercentile(attempt.TestID, attempt.TimeSpent)
	if err != nil {
		timePercentile = 0
	}
	
	// Определяем категорию распределения.
	distributionCategory := c.determineDistributionCategory(attempt.Percentage, percentileRank)
	
	// Определяем уровень навыка.
	skillLevel := c.determineSkillLevel(attempt.Percentage, percentileRank)
	
	// Собираем статистику по типам вопросов.
	byQuestionType, _ := c.aggregateQuestionTypeStats(userAnswers)
	
	// Генерируем рекомендации.
	recommendations := c.generateRecommendations(byQuestionType, attempt.Percentage)
	
	// Формируем итоговый анализ.
	analysis := &entities.DetailedAnalysis{
		AttemptID:              attemptID,
		TestID:                attempt.TestID,
		UserHash:              attempt.UserHash,
		Score:                 attempt.Score,
		MaxScore:              attempt.MaxScore,
		Percentage:            attempt.Percentage,
		TimeSpent:             attempt.TimeSpent,
		CreatedAt:             attempt.CreatedAt,
		PercentileRank:        percentileRank,
		TimePercentile:        timePercentile,
		DistributionCategory:  distributionCategory,
		SkillLevel:            skillLevel,
		TestStats:             testStats,
		ByQuestionType:        byQuestionType,
		Recommendations:       recommendations,
	}
	
	return analysis, nil
}

// GetTimePercentile возвращает перцентиль по времени.
func (c *Client) GetTimePercentile(testID string, timeSpent int) (float64, error) {
	var fasterAttempts int64
	err := c.db.Model(&entities.Attempt{}).
		Where("test_id = ? AND is_valid = ? AND time_spent < ?", 
			testID, true, timeSpent).
		Count(&fasterAttempts).Error
	
	if err != nil {
		return 0, err
	}
	
	var totalAttempts int64
	err = c.db.Model(&entities.Attempt{}).
		Where("test_id = ? AND is_valid = ?", testID, true).
		Count(&totalAttempts).Error
	
	if err != nil || totalAttempts == 0 {
		return 0, err
	}
	
	// Чем меньше время, тем лучше, поэтому считаем обратный перцентиль.
	timePercentile := (float64(fasterAttempts) + 0.5) / float64(totalAttempts) * 100
	if timePercentile > 100 {
		timePercentile = 100
	}
	
	return timePercentile, nil
}

// determineDistributionCategory определяет категорию распределения.
func (c *Client) determineDistributionCategory(percentage float64, percentile float64) *entities.DistributionCategory {
	categories := []entities.DistributionCategory{
		{
			Name:        "elite",
			Description: "Элита - топ 1% результатов",
			MinScore:    95.0,
			MaxScore:    100.0,
		},
		{
			Name:        "excellent",
			Description: "Отлично - топ 10%",
			MinScore:    85.0,
			MaxScore:    95.0,
		},
		{
			Name:        "good",
			Description: "Хорошо - выше среднего (топ 30%)",
			MinScore:    70.0,
			MaxScore:    85.0,
		},
		{
			Name:        "average",
			Description: "Средний результат",
			MinScore:    50.0,
			MaxScore:    70.0,
		},
		{
			Name:        "below_average",
			Description: "Ниже среднего",
			MinScore:    30.0,
			MaxScore:    50.0,
		},
		{
			Name:        "needs_improvement",
			Description: "Требует улучшения",
			MinScore:    0.0,
			MaxScore:    30.0,
		},
	}
	
	for _, category := range categories {
		if percentage >= category.MinScore && percentage < category.MaxScore {
			if percentile > 90 && category.Name != "elite" {
				return &categories[0]
			}
			if percentile < 10 && category.Name != "needs_improvement" {
				return &categories[5]
			}
			return &category
		}
	}
	
	// По умолчанию возвращаем среднюю категорию.
	return &categories[3]
}

// determinePerformanceQuadrant определяет эффективность прохождения.
func (c *Client) determinePerformanceQuadrant(percentage float64, timePercentile float64) *entities.PerformanceQuadrant {
	// Нормализуем значения.
	accuracy := percentage / 100.0
	speed := timePercentile / 100.0
	
	if accuracy >= 0.8 && speed <= 0.2 {
		return &entities.PerformanceQuadrant{
			Quadrant:    "master",
			Title:       "Мастер",
			Description: "Быстро и правильно. Топ-знания",
			XPosition:   accuracy * 100,
			YPosition:   (1 - speed) * 100,
		}
	}
	
	if accuracy >= 0.8 && speed > 0.2 {
		return &entities.PerformanceQuadrant{
			Quadrant:    "precise_but_slow",
			Title:       "Точно, но медленно",
			Description: "Высокий результат, но медленно",
			XPosition:   accuracy * 100,
			YPosition:   (1 - speed) * 100,
		}
	}
	
	if accuracy < 0.8 && speed <= 0.2 {
		return &entities.PerformanceQuadrant{
			Quadrant:    "fast_but_inaccurate",
			Title:       "Быстро, но неточно",
			Description: "Быстро и много ошибок",
			XPosition:   accuracy * 100,
			YPosition:   (1 - speed) * 100,
		}
	}

	return &entities.PerformanceQuadrant{
		Quadrant:    "needs_practice",
		Title:       "Требуется практика",
		Description: "Плоховато",
		XPosition:   accuracy * 100,
		YPosition:   (1 - speed) * 100,
	}
}

// determineSkillLevel определяет уровень.
func (c *Client) determineSkillLevel(percentage float64, percentile float64) string {
	if percentile >= 95 {
		return "Эксперт"
	}
	if percentile >= 80 {
		return "Продвинутый"
	}
	if percentile >= 50 {
		return "Средний"
	}
	if percentile >= 20 {
		return "Начинающий"
	}
	return "Новичок"
}

// aggregateQuestionTypeStats агрегирует статистику по типам вопросов.
func (c *Client) aggregateQuestionTypeStats(userAnswers []entities.UserAnswer) (map[string]entities.QuestionTypeStats, map[string]int) {
	typeStats := make(map[string]entities.QuestionTypeStats)
	timePerType := make(map[string]int)
	
	for _, ua := range userAnswers {
		question, err := c.GetQuestion(ua.QuestionID)
		if err != nil {
			continue
		}
		
		qType := question.QuestionType
		stats := typeStats[qType]
		
		stats.TotalQuestions++
		if ua.IsCorrect {
			stats.CorrectAnswers++
			stats.PointsEarned += ua.PointsEarned
		}
		stats.MaxPoints += question.Points
		
		typeStats[qType] = stats
	}
	
	// Рассчитываем проценты.
	for qType, stats := range typeStats {
		if stats.TotalQuestions > 0 {
			stats.Percentage = float64(stats.CorrectAnswers) / float64(stats.TotalQuestions) * 100
			typeStats[qType] = stats
		}
	}
	
	return typeStats, timePerType
}

// generateRecommendations генерирует рекомендации.
func (c *Client) generateRecommendations(byQuestionType map[string]entities.QuestionTypeStats, overallPercentage float64) []entities.Recommendation {
	var recommendations []entities.Recommendation
	
	// Рекомендации на основе общего процента.
	if overallPercentage < 50 {
		recommendations = append(recommendations, entities.Recommendation{
			ID:          "general_low_score",
			Title:       "Повторите основы",
			Description: "Ваш результат ниже 50%. Нужно подтянуться в теме.",
			Priority:    "high",
		})
	}
	
	// Рекомендации на основе типов вопросов.
	for qType, stats := range byQuestionType {
		if stats.Percentage < 60 && stats.TotalQuestions >= 3 {
			priority := "medium"
			if stats.Percentage < 40 {
				priority = "high"
			}
			
			title := fmt.Sprintf("Улучшить надо: %s", getQuestionTypeName(qType))
			
			recommendations = append(recommendations, entities.Recommendation{
				ID:          fmt.Sprintf("improve_%s", qType),
				Title:       title,
				Description: fmt.Sprintf("Точность по типу вопросов '%s': %.1f%%", getQuestionTypeName(qType), stats.Percentage),
				Priority:    priority,
				QuestionTypes: []string{qType},
			})
		}
	}
	
	// Рекомендация по скорости.
	if len(recommendations) == 0 && overallPercentage > 80 {
		recommendations = append(recommendations, entities.Recommendation{
			ID:          "speed_improvement",
			Title:       "Попробуйте пройти быстрее",
			Description: "Ваша точность отличная! Можно побыстрее только.",
			Priority:    "low",
		})
	}
	
	return recommendations
}

// getQuestionTypeName возвращает читаемое имя типа вопроса.
func getQuestionTypeName(qType string) string {
	names := map[string]string{
		"flag":     "Флаги",
		"capital":  "Столицы",
		"region":   "Регионы",
	}
	
	if name, ok := names[qType]; ok {
		return name
	}
	return qType
}

// МЕТОДЫ ДЛЯ ТЕСТИРОВАНИЯ. ПОТОМ УБРАТЬ НАДО.

// CreateTestData создает тестовые данные.
func (c *Client) CreateTestData() error {
	log.Println("=== Creating comprehensive test data ===")
	
	// Создаем тесты если их нет.
	log.Println("Creating/updating tests...")
	if err := c.createTests(); err != nil {
		log.Printf("Warning: CreateTests failed: %v", err)
	}
	
	log.Println("Cleaning old data...")
	c.db.Exec("DELETE FROM attempts")
	c.db.Exec("DELETE FROM test_stats")
	
	// Создаем тестовые попытки для каждого теста.
	tests := []string{"europe", "asia", "world"}
	
	for _, testID := range tests {
		log.Printf("Creating 100 attempts for %s...", testID)
		
		// Параметры для каждого теста.
		var meanPercentage, meanTime float64
		
		switch testID {
		case "europe":
			meanPercentage = 65.0
			meanTime = 180.0
		case "asia":
			meanPercentage = 70.0
			meanTime = 160.0
		case "world":
			meanPercentage = 60.0
			meanTime = 220.0
		default:
			meanPercentage = 65.0
			meanTime = 180.0
		}
		
		// Создаем 100 попыток с реалистичным распределением.
		for i := 0; i < 100; i++ {
			// Генерация процента.
			percentage := meanPercentage + (rand.Float64()-0.5)*30
			if percentage < 0 {
				percentage = 0
			}
			if percentage > 100 {
				percentage = 100
			}
			
			// Генерация времени.
			timeSpent := int(meanTime + (rand.Float64()-0.5)*120)
			if timeSpent < 30 {
				timeSpent = 30
			}
			if timeSpent > 600 {
				timeSpent = 600
			}
			
			// Валидные попытки.
			isValid := rand.Float32() > 0.1
			
			attempt := &entities.Attempt{
				ID:         uuid.New().String(),
				TestID:     testID,
				UserHash:   fmt.Sprintf("user_%d", i%50),
				Score:      int(percentage),
				MaxScore:   100,
				Percentage: percentage,
				TimeSpent:  timeSpent,
				Answers:    "{}",
				IsValid:    isValid,
				CreatedAt:  time.Now().Add(-time.Duration(i*6) * time.Hour),
			}
			
			if err := c.db.Create(attempt).Error; err != nil {
				if !strings.Contains(err.Error(), "Duplicate entry") {
					log.Printf("Error creating attempt: %v", err)
				}
			}
		}
		
		log.Printf("Created 100 attempts for %s", testID)
	}
	
	// Пересчитываем статистику.
	log.Println("Recalculating statistics...")
	if err := c.recalculateAllStats(); err != nil {
		log.Printf("Error recalculating stats: %v", err)
	}
	
	log.Println("=== Test data creation completed ===")
	return nil
}

func (c *Client) createTests() error {
	tests := []entities.Test{
		{
			ID:   1,
			Type: 1,
			Name: "europe",
			I18n: entities.JSON{
				"en": map[string]interface{}{"desc": "European countries test"},
				"ru": map[string]interface{}{"desc": "Тест на знание стран Европы"},
			},
			Settings: entities.JSON{
				"modes":        []string{"flags", "capitals", "regions"},
				"width":        1529,
				"height":       843,
				"international": true,
			},
		},
		{
			ID:   2,
			Type: 1,
			Name: "asia",
			I18n: entities.JSON{
				"en": map[string]interface{}{"desc": "Asian countries test"},
				"ru": map[string]interface{}{"desc": "Тест на знание стран Азии"},
			},
			Settings: entities.JSON{
				"modes":        []string{"flags", "capitals"},
				"width":        1529,
				"height":       843,
			},
		},
		{
			ID:   3,
			Type: 1,
			Name: "world",
			I18n: entities.JSON{
				"en": map[string]interface{}{"desc": "World countries test"},
				"ru": map[string]interface{}{"desc": "Тест на знание стран мира"},
			},
			Settings: entities.JSON{
				"modes":        []string{"flags", "capitals"},
				"width":        1529,
				"height":       843,
			},
		},
	}
	
	for _, test := range tests {
		result := c.db.Save(&test)
		if result.Error != nil {
			if strings.Contains(result.Error.Error(), "Duplicate entry") {
				continue
			}
			return result.Error
		}
	}
	
	return nil
}

func (c *Client) recalculateAllStats() error {
	log.Println("Starting statistics recalculation...")
	
	var testIDs []string
	err := c.db.Table("attempts").Select("DISTINCT test_id").Find(&testIDs).Error
	if err != nil {
		return err
	}
	
	if len(testIDs) == 0 {
		log.Println("No tests found in attempts table")
		return nil
	}
	
	for _, testID := range testIDs {
		log.Printf("Recalculating stats for test: %s", testID)
		
		if err := c.recalculateStatsForTest(testID); err != nil {
			log.Printf("Error recalculating stats for %s: %v", testID, err)
			continue
		}
	}
	
	log.Println("Statistics recalculation completed")
	return nil
}

// recalculateStatsForTest - пересчитывает статистику для конкретного теста.
func (c *Client) recalculateStatsForTest(testID string) error {
	// Агрегируем данные из attempts.
	var result struct {
		TotalAttempts   int64
		ValidAttempts   int64
		SumPercentage   float64
		SumTimeSpent    float64
	}
	
	// Запрос для агрегации
	err := c.db.Table("attempts").
		Select(`COUNT(*) as total_attempts,
			SUM(CASE WHEN is_valid = true THEN 1 ELSE 0 END) as valid_attempts,
			SUM(CASE WHEN is_valid = true THEN percentage ELSE 0 END) as sum_percentage,
			SUM(CASE WHEN is_valid = true THEN time_spent ELSE 0 END) as sum_time_spent`).
		Where("test_id = ?", testID).
		Scan(&result).Error
	
	if err != nil {
		return fmt.Errorf("aggregation query failed: %w", err)
	}
	
	// Вычисляем средние
	avgPercentage := 0.0
	avgTimeSpent := 0.0
	if result.ValidAttempts > 0 {
		avgPercentage = result.SumPercentage / float64(result.ValidAttempts)
		avgTimeSpent = result.SumTimeSpent / float64(result.ValidAttempts)
	}
	
	log.Printf("Calculated for %s: %d total, %d valid, avg: %.1f%%, avg time: %.1fs",
		testID, result.TotalAttempts, result.ValidAttempts, avgPercentage, avgTimeSpent)
	
	// Находим или создаем запись статистики
	var stats entities.TestStats
	err = c.db.Where("test_id = ?", testID).First(&stats).Error
	
	if err == gorm.ErrRecordNotFound {
		// Создаем новую запись
		stats = entities.TestStats{
			ID:            uuid.New().String(),
			TestID:        testID,
			Date:          time.Now(),
			TotalAttempts: int(result.TotalAttempts),
			ValidAttempts: int(result.ValidAttempts),
			AvgPercentage: avgPercentage,
			AvgTimeSpent:  avgTimeSpent,
		}
		
		if err := c.db.Create(&stats).Error; err != nil {
			return fmt.Errorf("failed to create stats: %w", err)
		}
		
		log.Printf("Created new stats for %s", testID)
		
	} else if err != nil {
		return fmt.Errorf("failed to find stats: %w", err)
		
	} else {
		// Обновляем существующую запись
		stats.Date = time.Now()
		stats.TotalAttempts = int(result.TotalAttempts)
		stats.ValidAttempts = int(result.ValidAttempts)
		stats.AvgPercentage = avgPercentage
		stats.AvgTimeSpent = avgTimeSpent
		
		if err := c.db.Save(&stats).Error; err != nil {
			return fmt.Errorf("failed to update stats: %w", err)
		}
		
		log.Printf("Updated stats for %s", testID)
	}
	
	return nil
}