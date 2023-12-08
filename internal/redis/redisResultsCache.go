package redis

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/fe-dox/tc-pbn-extractor/internal/data"
	"github.com/redis/go-redis/v9"
	"time"
)

const PROCESSING = "processing"

type ResultsCache struct {
	rdb *redis.Client
}

func (r ResultsCache) Get(key string) (data.JobStatus, data.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	strResult, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return data.JobNotFound, data.Result{}, nil
		}
		return data.JobNotFound, data.Result{}, err
	}
	if strResult == PROCESSING {
		return data.JobProcessing, data.Result{}, err
	}
	var result data.Result
	err = json.Unmarshal([]byte(strResult), &result)
	if err != nil {
		return 0, data.Result{}, err
	}
	return data.JobDone, result, nil
}

func (r ResultsCache) GetStatus(key string) (data.JobStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	strResult, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return data.JobNotFound, nil
		}
		return data.JobNotFound, err
	}
	if strResult == PROCESSING {
		return data.JobProcessing, err
	}
	return data.JobDone, nil
}

func (r ResultsCache) SaveResult(key string, value data.Result) error {
	parsedData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	_, err = r.rdb.Set(ctx, key, string(parsedData), time.Minute*15).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r ResultsCache) SetStatusProcessing(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	_, err := r.rdb.Set(ctx, key, PROCESSING, time.Minute*5).Result()
	if err != nil {
		return err
	}
	return nil
}

func NewResultsCache(connectionUrl string) (*ResultsCache, error) {
	options, err := redis.ParseURL(connectionUrl)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(options)
	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	return &ResultsCache{rdb: rdb}, nil
}
