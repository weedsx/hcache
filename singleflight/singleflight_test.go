package singleflight

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// 模拟的函数，用于测试
func mockFunc(result string, duration time.Duration, err error) func() (any, error) {
	return func() (any, error) {
		time.Sleep(duration) // 模拟函数执行时间
		return result, err
	}
}

func TestGroup_Do_Success(t *testing.T) {
	var g Group
	key := "testKey"
	expectedResult := "result"

	// 启动多个并发请求，模拟高并发场景
	var wg sync.WaitGroup
	numGoroutines := 10
	wg.Add(numGoroutines)
	results := make([]any, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			val, err := g.Do(key, mockFunc(expectedResult, 100*time.Millisecond, nil))
			results[i] = val
			errors[i] = err
		}(i)
	}

	wg.Wait()

	// 验证结果
	for i := 0; i < numGoroutines; i++ {
		if results[i] != expectedResult {
			t.Errorf("Expected %v, but got %v", expectedResult, results[i])
		}
		if errors[i] != nil {
			t.Errorf("Expected nil error, but got %v", errors[i])
		}
	}
}

func TestGroup_Do_Error(t *testing.T) {
	var g Group
	key := "testKey"
	expectedErr := errors.New("mock error")

	// 启动多个并发请求，模拟高并发场景
	var wg sync.WaitGroup
	numGoroutines := 5
	wg.Add(numGoroutines)
	results := make([]any, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			val, err := g.Do(key, mockFunc("", 100*time.Millisecond, expectedErr))
			results[i] = val
			errors[i] = err
		}(i)
	}

	wg.Wait()

	// 验证错误
	for i := 0; i < numGoroutines; i++ {
		if results[i] != "" {
			t.Errorf("Expected empty result, but got %v", results[i])
		}
		if errors[i] != expectedErr {
			t.Errorf("Expected error %v, but got %v", expectedErr, errors[i])
		}
	}
}
