package listener

import (
	"context"
	"reflect"
	"sort"
	"sync"

	"github.com/SUPERDBFMP/go-base/glog"
)

// ========== 事件发布器 ==========

// EventPublisher 负责注册监听器及发布事件
type EventPublisher struct {
	listeners []TypedApplicationListener[ApplicationEvent]
	mutex     sync.RWMutex
	once      sync.Once
}

var globalEventPublisher = &EventPublisher{}

// init 确保初始化
func (ep *EventPublisher) init() {
	ep.once.Do(func() {
		if ep.listeners == nil {
			ep.listeners = make([]TypedApplicationListener[ApplicationEvent], 0)
		}
	})
}

// AddListener 添加监听器（兼容原有接口）
func (ep *EventPublisher) addListener(listener TypedApplicationListener[ApplicationEvent]) {
	ep.init()
	ep.mutex.Lock()
	defer ep.mutex.Unlock()
	ep.listeners = append(ep.listeners, listener)
	sort.Slice(
		ep.listeners, func(i, j int) bool {
			return ep.listeners[i].GetOrder() < ep.listeners[j].GetOrder()
		},
	)
}

// AddTypedListener 添加泛型监听器（包级函数）
func addTypedListener[T ApplicationEvent](publisher *EventPublisher, listener TypedApplicationListener[T]) {
	wrapper := &genericListenerWrapper[T]{
		listener: listener,
	}
	publisher.addListener(wrapper)
}

// PublishEvent 发布事件给所有监听器
func (ep *EventPublisher) PublishEvent(ctx context.Context, event ApplicationEvent) {
	ep.init()
	defer func() {
		if r := recover(); r != nil {
			glog.Errorf(ctx, "event listener panic recovered: %v", r)
		}
	}()
	for _, listener := range ep.listeners {
		if event.SupportAsync() {
			go func(l TypedApplicationListener[ApplicationEvent]) {
				defer func() {
					if r := recover(); r != nil {
						glog.Errorf(ctx, "event listener panic recovered: %v", r)
					}
				}()
				l.OnApplicationEvent(ctx, event)
			}(listener)
		} else {
			listener.OnApplicationEvent(ctx, event)
		}
	}
}

// ========== 全局便捷函数 ==========

// AddTypedApplicationListener 添加泛型监听器的全局函数
func AddTypedApplicationListener[T ApplicationEvent](listener TypedApplicationListener[T]) {
	addTypedListener(globalEventPublisher, listener)
}

// PublishApplicationEvent 发布事件的全局函数
func PublishApplicationEvent(ctx context.Context, event ApplicationEvent) {
	glog.Infof(ctx, "publish eventName: %v,event:%v", reflect.TypeOf(event).String(), event)
	globalEventPublisher.PublishEvent(ctx, event)
}
