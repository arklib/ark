package retry

import (
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/arklib/ark/queue"
)

type DBRetryDriver struct {
	queue.RetryDriver
	db *gorm.DB
}

type QueueRetry struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Topic     string    `json:"topic" gorm:"index:idx"`
	Task      string    `json:"task" gorm:"index:idx"`
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

func (r *DBRetryDriver) Init(topic, task string) error {
	hasTable := r.db.Migrator().HasTable("queue_retry")
	if !hasTable {
		return r.db.AutoMigrate(&QueueRetry{})
	}
	return nil
}

func (r *DBRetryDriver) Add(topic, task string, rawMessage []byte, errMessage string, interval uint, isFailed bool) error {
	second := time.Duration(interval) * time.Second

	item := &QueueRetry{
		Topic:    topic,
		Task:     task,
		Message:  string(rawMessage),
		Error:    errMessage,
		IsFailed: isFailed,
		Interval: interval,
		NextAt:   time.Now().Add(second),
	}
	return r.db.Create(item).Error
}

func (r *DBRetryDriver) Run(topic string, task string, push queue.RetryPush) error {
	page := 1
	for {
		var list []QueueRetry

		pageSize := 100
		offset := (page - 1) * pageSize
		result := r.db.
			Where("topic = ?", topic).
			Where("task = ?", task).
			Where("is_failed = ?", false).
			Where("next_at < ?", time.Now()).
			Order("id asc").
			Offset(offset).
			Limit(pageSize).
			Find(&list)

		if result.RowsAffected == 0 {
			return nil
		}

		for _, item := range list {
			id := strconv.Itoa(int(item.ID))
			err := push(id, []byte(item.Message))
			if err != nil {
				return err
			}
			r.db.Delete(&item)
		}
		page++
	}
}
