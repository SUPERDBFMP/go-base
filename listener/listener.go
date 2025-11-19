package listener

import (
	"context"
)

// ApplicationEvent 代表应用程序生命周期中的一个事件
type ApplicationEvent interface {
	SupportAsync() bool
}

// ========== 泛型监听器实现 ==========

// TypedApplicationListener 泛型监听器接口
type TypedApplicationListener[T ApplicationEvent] interface {
	OnApplicationEvent(ctx context.Context, event T)
	GetOrder() int
}

// genericListenerWrapper 泛型监听器包装器
type genericListenerWrapper[T ApplicationEvent] struct {
	listener TypedApplicationListener[T]
}

func (w *genericListenerWrapper[T]) OnApplicationEvent(ctx context.Context, event ApplicationEvent) {
	if typedEvent, ok := event.(T); ok {
		w.listener.OnApplicationEvent(ctx, typedEvent)
	}
}

func (w *genericListenerWrapper[T]) GetOrder() int {
	return w.listener.GetOrder()
}
