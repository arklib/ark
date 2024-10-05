package queue

import (
	"strconv"
	"time"

	"gorm.io/gorm"
)

type DBRetryDriver struct {
	RetryDriver
	db *gorm.DB
}

type TaskRetry struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Topic     string    `json:"topic" gorm:"index:idx"`
	Name      string    `json:"name" gorm:"index:idx"`
	IsFailed  bool      `json:"isFailed" gorm:"index:idx"`
	Interval  uint      `json:"interval"`
	Message   string    `json:"message" gorm:"type:json"`
	Error     string    `json:"error" gorm:"type:text"`
	NextAt    time.Time `json:"nextAt" gorm:"index:idx"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewDBRetryDriver(db *gorm.DB) *DBRetryDriver {
	return &DBRetryDriver{db: db}
}

func (r *DBRetryDriver) Init(topic, name string) error {
	hasTable := r.db.Migrator().HasTable("task_retry")
	if !hasTable {
		return r.db.AutoMigrate(&TaskRetry{})
	}
	return nil
}

func (r *DBRetryDriver) Add(topic, name string, rawMessage []byte, errMessage string, interval uint, isFailed bool) error {
	second := time.Duration(interval) * time.Second

	task := &TaskRetry{
		Topic:    topic,
		Name:     name,
		Message:  string(rawMessage),
		Error:    errMessage,
		IsFailed: isFailed,
		Interval: interval,
		NextAt:   time.Now().Add(second),
	}
	return r.db.Create(task).Error
}

func (r *DBRetryDriver) Run(topic string, name string, push RetryPush) error {
	page := 1
	for {
		var tasks []TaskRetry

		pageSize := 100
		offset := (page - 1) * pageSize
		result := r.db.
			Where("topic = ?", topic).
			Where("name = ?", name).
			Where("is_failed = ?", false).
			Where("next_at < ?", time.Now()).
			Order("id asc").
			Offset(offset).
			Limit(pageSize).
			Find(&tasks)

		if result.RowsAffected == 0 {
			return nil
		}

		for _, task := range tasks {
			id := strconv.Itoa(int(task.ID))
			err := push(id, []byte(task.Message))
			if err != nil {
				return err
			}
			r.db.Delete(&task)
		}
		page++
	}
}
