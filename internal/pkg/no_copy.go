package pkg

// NoCopy noCopy 是一个特殊的类型，用于防止结构体被复制
type NoCopy struct{}

func (*NoCopy) Lock()   {}
func (*NoCopy) Unlock() {}
