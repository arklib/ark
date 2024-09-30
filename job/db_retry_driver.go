package job

import (
	"strconv"
	"time"

	"gorm.io/gorm"
)

type DBRetryDriver struct {
	RetryDriver
	db *gorm.DB
}

type JobRetry struct {
	Id        uint      `json:"id" gorm:"primaryKey"`
	Queue     string    `json:"queue" gorm:"type:varchar(64);index:idx_queue_time"`
	Time      time.Time `json:"time" gorm:"index:idx_queue_time"`
	Data      string    `json:"data" gorm:"type:text"`
	Error     string    `json:"error" gorm:"type:text"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewDBRetryDriver(db *gorm.DB) *DBRetryDriver {
	hasTable := db.Migrator().HasTable("job_retry")
	if !hasTable {
		_ = db.AutoMigrate(&JobRetry{})
	}
	return &DBRetryDriver{db: db}
}

func (r *DBRetryDriver) Init() error {
	hasTable := r.db.Migrator().HasTable("job_retry")
	if !hasTable {
		return r.db.AutoMigrate(&JobRetry{})
	}
	return nil
}

func (r *DBRetryDriver) Add(queue string, data []byte, retryTime uint, errMsg string) error {
	second := time.Duration(retryTime) * time.Second
	job := &JobRetry{
		Queue: queue,
		Data:  string(data),
		Error: errMsg,
		Time:  time.Now().Add(second),
	}
	return r.db.Create(job).Error
}

func (r *DBRetryDriver) Run(queue string, push PushCallback) error {
	page := 1
	for {
		var jobs []JobRetry

		pageSize := 100
		offset := (page - 1) * pageSize
		result := r.db.
			Where("queue = ?", queue).
			Where("time < ?", time.Now()).
			Order("id asc").
			Offset(offset).
			Limit(pageSize).
			Find(&jobs)

		if result.RowsAffected == 0 {
			return nil
		}

		for _, job := range jobs {
			id := strconv.Itoa(int(job.Id))
			rawData := []byte(job.Data)

			err := push(id, rawData)
			if err != nil {
				return err
			}
			r.db.Delete(&job)
		}
		page++
	}
}
