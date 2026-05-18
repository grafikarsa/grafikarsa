package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type CaptchaService struct {
	redis    *redis.Client
	inMemory map[string]captchaData
	mu       sync.RWMutex
}

func NewCaptchaService(redisClient *redis.Client) *CaptchaService {
	return &CaptchaService{
		redis:    redisClient,
		inMemory: make(map[string]captchaData),
	}
}

type captchaData struct {
	Answer    int    `json:"answer"`
	CreatedAt int64  `json:"created_at"`
}

func (s *CaptchaService) Generate(ctx context.Context) (id string, question string, err error) {
	id = generateID()
	a, b, answer := generateMathProblem()

	question = fmt.Sprintf("Berapa %d + %d?", a, b)

	data := captchaData{
		Answer:    answer,
		CreatedAt: time.Now().Unix(),
	}

	if s.redis != nil {
		encoded, _ := json.Marshal(data)
		err = s.redis.Set(ctx, "captcha:"+id, encoded, 5*time.Minute).Err()
		if err != nil {
			log.Printf("[captcha] Redis write failed, using in-memory store: %v", err)
			s.storeInMemory(id, data)
		}
	} else {
		s.storeInMemory(id, data)
	}

	return id, question, nil
}

func (s *CaptchaService) Verify(ctx context.Context, id string, answer int) (bool, error) {
	if s.redis != nil {
		val, err := s.redis.Get(ctx, "captcha:"+id).Result()
		if err == redis.Nil {
			return s.verifyInMemory(id, answer)
		}
		if err != nil {
			log.Printf("[captcha] Redis read failed: %v", err)
			return s.verifyInMemory(id, answer)
		}

		var data captchaData
		if err := json.Unmarshal([]byte(val), &data); err != nil {
			return false, fmt.Errorf("failed to parse captcha: %w", err)
		}

		s.redis.Del(ctx, "captcha:"+id)

		if time.Since(time.Unix(data.CreatedAt, 0)) > 5*time.Minute {
			return false, nil
		}

		return data.Answer == answer, nil
	}

	return s.verifyInMemory(id, answer)
}

func (s *CaptchaService) TrackFailedLogin(ctx context.Context, key string) error {
	if s.redis == nil {
		return nil
	}

	pipe := s.redis.Pipeline()
	pipe.Incr(ctx, "login:failed:"+key)
	pipe.Expire(ctx, "login:failed:"+key, 5*time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *CaptchaService) GetFailedCount(ctx context.Context, key string) (int, error) {
	if s.redis == nil {
		return 0, nil
	}

	val, err := s.redis.Get(ctx, "login:failed:"+key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		log.Printf("[captcha] Redis read failed: %v", err)
		return 0, nil
	}

	count, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *CaptchaService) ResetFailedLogin(ctx context.Context, key string) error {
	if s.redis == nil {
		return nil
	}
	return s.redis.Del(ctx, "login:failed:"+key).Err()
}

func (s *CaptchaService) storeInMemory(id string, data captchaData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inMemory[id] = data

	go func() {
		time.Sleep(5 * time.Minute)
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.inMemory, id)
	}()
}

func (s *CaptchaService) verifyInMemory(id string, answer int) (bool, error) {
	s.mu.RLock()
	data, ok := s.inMemory[id]
	s.mu.RUnlock()

	if !ok {
		return false, nil
	}

	s.mu.Lock()
	delete(s.inMemory, id)
	s.mu.Unlock()

	if time.Since(time.Unix(data.CreatedAt, 0)) > 5*time.Minute {
		return false, nil
	}

	return data.Answer == answer, nil
}

func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func generateMathProblem() (a, b, answer int) {
	a = rand.Intn(10) + 1
	b = rand.Intn(10) + 1
	answer = a + b
	return
}
