// application/scheduler/scheduler.go
package scheduler

import (
	"context"
	"sync"
	"time"

	"crypto-exchange-screener-bot/pkg/logger"
)

// Schedule определяет расписание задачи
type Schedule struct {
	// DailyAt: задача запускается раз в день в заданное UTC время
	// Every: задача запускается с заданным интервалом
	kind     scheduleKind
	hour     int
	minute   int
	interval time.Duration
}

type scheduleKind int

const (
	kindDaily    scheduleKind = iota // раз в сутки в HH:MM UTC
	kindInterval                     // каждые N единиц времени
)

// DailyAt создает расписание "каждый день в HH:MM UTC"
func DailyAt(hour, minute int) Schedule {
	return Schedule{kind: kindDaily, hour: hour, minute: minute}
}

// Every создает расписание "каждые N времени"
func Every(d time.Duration) Schedule {
	return Schedule{kind: kindInterval, interval: d}
}

// nextRun вычисляет время следующего запуска относительно now
func (s Schedule) nextRun(now time.Time) time.Time {
	switch s.kind {
	case kindDaily:
		next := time.Date(now.Year(), now.Month(), now.Day(), s.hour, s.minute, 0, 0, time.UTC)
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		return next
	case kindInterval:
		return now.Add(s.interval)
	default:
		return now.Add(24 * time.Hour)
	}
}

// Job описывает одну планируемую задачу
type Job struct {
	Name        string
	Description string
	Schedule    Schedule
	Handler     func(ctx context.Context) error

	mu      sync.Mutex
	nextRun time.Time
	lastRun time.Time
	lastErr error
	runs    int
}

// Status возвращает текущее состояние задачи
func (j *Job) Status() JobStatus {
	j.mu.Lock()
	defer j.mu.Unlock()
	return JobStatus{
		Name:        j.Name,
		Description: j.Description,
		NextRun:     j.nextRun,
		LastRun:     j.lastRun,
		LastErr:     j.lastErr,
		Runs:        j.runs,
	}
}

// JobStatus снапшот состояния задачи
type JobStatus struct {
	Name        string
	Description string
	NextRun     time.Time
	LastRun     time.Time
	LastErr     error
	Runs        int
}

// Scheduler управляет всеми cron-задачами приложения
type Scheduler struct {
	jobs     []*Job
	mu       sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// New создает новый планировщик
func New() *Scheduler {
	return &Scheduler{
		stopChan: make(chan struct{}),
	}
}

// Register добавляет задачу в планировщик.
// Должен вызываться до Start().
func (s *Scheduler) Register(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job.nextRun = job.Schedule.nextRun(time.Now().UTC())
	s.jobs = append(s.jobs, job)

	logger.Info("📋 [Scheduler] Зарегистрирована задача %q — первый запуск в %s",
		job.Name, job.nextRun.Format("2006-01-02 15:04:05 UTC"))
}

// Start запускает цикл планировщика в фоновой горутине
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.loop()
	}()
	logger.Info("✅ [Scheduler] Запущен (%d задач)", len(s.jobs))
}

// Stop останавливает планировщик и ждёт завершения текущих задач
func (s *Scheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
	logger.Info("🛑 [Scheduler] Остановлен")
}

// Jobs возвращает статус всех задач
func (s *Scheduler) Jobs() []JobStatus {
	s.mu.RLock()
	jobs := make([]*Job, len(s.jobs))
	copy(jobs, s.jobs)
	s.mu.RUnlock()

	statuses := make([]JobStatus, len(jobs))
	for i, j := range jobs {
		statuses[i] = j.Status()
	}
	return statuses
}

// loop — основной цикл: каждую минуту проверяет, какие задачи нужно запустить
func (s *Scheduler) loop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Первая проверка сразу при старте
	s.tick()

	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-s.stopChan:
			return
		}
	}
}

// tick проверяет все задачи и запускает те, у которых наступило время
func (s *Scheduler) tick() {
	now := time.Now().UTC()

	s.mu.RLock()
	jobs := make([]*Job, len(s.jobs))
	copy(jobs, s.jobs)
	s.mu.RUnlock()

	for _, job := range jobs {
		job.mu.Lock()
		due := !now.Before(job.nextRun)
		job.mu.Unlock()

		if due {
			s.wg.Add(1)
			go s.run(job)
		}
	}
}

// run выполняет одну задачу и обновляет её состояние
func (s *Scheduler) run(job *Job) {
	defer s.wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger.Info("▶️  [Scheduler] Запуск задачи %q", job.Name)
	start := time.Now()

	err := job.Handler(ctx)

	elapsed := time.Since(start)

	job.mu.Lock()
	job.lastRun = start
	job.lastErr = err
	job.runs++
	job.nextRun = job.Schedule.nextRun(time.Now().UTC())
	nextRun := job.nextRun
	job.mu.Unlock()

	if err != nil {
		logger.Error("❌ [Scheduler] Задача %q завершилась с ошибкой за %v: %v", job.Name, elapsed, err)
	} else {
		logger.Info("✅ [Scheduler] Задача %q выполнена за %v. Следующий запуск: %s",
			job.Name, elapsed, nextRun.Format("2006-01-02 15:04:05 UTC"))
	}
}
