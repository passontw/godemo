package rocketmqclient

import "errors"

// ErrAlreadyExists 表示資源已存在
var ErrAlreadyExists = errors.New("resource already exists")

// ErrNotFound 表示資源不存在
var ErrNotFound = errors.New("resource not found")

// ErrInvalidArgument 表示參數不合法
var ErrInvalidArgument = errors.New("invalid argument")

// ErrRequestTimeout 表示請求超時
var ErrRequestTimeout = errors.New("request timeout")

// ErrRequestFailed 表示請求失敗
var ErrRequestFailed = errors.New("request failed")

// 可擴充自訂錯誤型別，並於 client.go 包裝底層錯誤
