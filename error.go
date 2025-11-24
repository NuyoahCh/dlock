package dlock

import "errors"

var (
	// ErrLockNotHold 未持有锁，进行操作
	ErrLockNotHold = errors.New("dlock: 未持有锁")
	// ErrLocked 加锁时发现，锁被持有
	ErrLocked = errors.New("dlock: 加锁失败，锁被人持有")
)
