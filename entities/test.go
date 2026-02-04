package entities

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Test.
type Test struct {
	ID       int    `json:"id" gorm:"column:id;primaryKey;autoIncrement:false"`
	Type     int    `json:"type" gorm:"column:type"`
	Name     string `json:"name" gorm:"column:name;unique"`
	I18n     JSON   `json:"i18n" gorm:"column:i18n;type:json"`
	Settings JSON   `json:"settings" gorm:"column:settings;type:json"`
}

// Attempt - попытка прохождения теста.
type Attempt struct {
	ID         string    `json:"id" gorm:"primaryKey;column:id"`
	TestID     string    `json:"test_id" gorm:"column:test_id"`
	VersionID  string    `json:"version_id" gorm:"column:version_id"`
	UserHash   string    `json:"user_hash" gorm:"column:user_hash"`
	Score      int       `json:"score" gorm:"column:score"`
	MaxScore   int       `json:"max_score" gorm:"column:max_score"`
	Percentage float64   `json:"percentage" gorm:"column:percentage"`
	TimeSpent  int       `json:"time_spent" gorm:"column:time_spent"`
	Answers    string    `json:"answers" gorm:"column:answers;type:text"`
	IsValid    bool      `json:"is_valid" gorm:"column:is_valid;default:true"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at"`
}

type Question struct {
	ID           string `json:"id" gorm:"column:id;primaryKey"`
	TestID       int    `json:"test_id" gorm:"column:test_id"`
	QuestionType string `json:"question_type" gorm:"column:question_type"`
	QuestionText string `json:"question_text" gorm:"column:question_text"`
	CorrectAnswer string `json:"correct_answer" gorm:"column:correct_answer"`
	Options      JSON   `json:"options" gorm:"column:options;type:json"`
	Points       int    `json:"points" gorm:"column:points;default:1"`
	Metadata     JSON   `json:"metadata" gorm:"column:metadata;type:json"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
}

type UserAnswer struct {
	ID          string    `json:"id" gorm:"column:id;primaryKey"`
	AttemptID   string    `json:"attempt_id" gorm:"column:attempt_id"`
	QuestionID  string    `json:"question_id" gorm:"column:question_id"`
	UserAnswer  string    `json:"user_answer" gorm:"column:user_answer"`
	IsCorrect   bool      `json:"is_correct" gorm:"column:is_correct"`
	PointsEarned int      `json:"points_earned" gorm:"column:points_earned"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
}

// TestStats - статистика по тесту.
type TestStats struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	TestID        string    `json:"test_id" gorm:"index"`
	Date          time.Time `json:"date"`
	TotalAttempts int       `json:"total_attempts"`
	ValidAttempts int       `json:"valid_attempts"`
	AvgPercentage float64   `json:"avg_percentage"`
	AvgTimeSpent  float64   `json:"avg_time_spent"`
}

// AttemptResult - результат проверки.
type AttemptResult struct {
	AttemptID  string           `json:"attempt_id"`
	TestID     string           `json:"test_id"`
	VersionID  string           `json:"version_id"`
	UserHash   string           `json:"user_hash"`
	Score      int              `json:"score"`
	MaxScore   int              `json:"max_score"`
	Percentage float64          `json:"percentage"`
	Details    []QuestionResult `json:"details"`
}

type QuestionResult struct {
	QuestionID string `json:"question_id"`
	Text       string `json:"text"`
	UserAnswer string `json:"user_answer"`
	IsCorrect  bool   `json:"is_correct"`
	Score      int    `json:"score"`
	MaxScore   int    `json:"max_score"`
}

// JSON для работы с JSON полями
type JSON map[string]interface{}

func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	
	return json.Unmarshal(b, &j)
}