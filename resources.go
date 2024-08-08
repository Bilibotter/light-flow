package light_flow

import (
	"sync"
)

var resourceManagers = make(map[string]*resourceManager)
var resourceLock = new(sync.RWMutex)

var resPrefix = []byte("*")

type Resource interface {
	runtimeI
	Put(key string, value any) any
	Fetch(key string) (value any, exist bool)
}

type resCtx interface {
	runtimeI
	sync.Locker
}

type ResourceManager interface {
	OnInitialize(func(res Resource, initParam any) (resInstance any, err error)) ResourceManager
	OnRecover(func(res Resource) error) ResourceManager
	OnSuspend(func(res Resource) error) ResourceManager
	OnRelease(func(res Resource) error) ResourceManager
}

type resourceManager struct {
	name         string
	onInitialize func(res Resource, initParam any) (resInstance any, err error)
	onRecover    func(res Resource) error
	onSuspend    func(res Resource) error
	onRelease    func(res Resource) error
}

type resource struct {
	resCtx
	ctx      map[string]any
	instance any
}

type resSerializable struct {
	Ctx      map[string]any
	Instance any
}

func RegisterResourceManager(name string) ResourceManager {
	if !isValidIdentifier(name) {
		panic(patternHint)
	}
	resourceLock.Lock()
	defer resourceLock.Unlock()
	if _, exist := resourceManagers[name]; exist {
		panic("resource manager already registered: " + name)
	}
	foo := &resourceManager{
		name:      name,
		onRecover: emptyHandler,
		onSuspend: emptyHandler,
		onRelease: emptyHandler,
	}
	resourceManagers[name] = foo
	return foo
}

func getResourceManager(name string) (foo *resourceManager, exist bool) {
	foo, exist = resourceManagers[name]
	return
}

func emptyHandler(_ Resource) error {
	return nil
}

func (rm *resourceManager) OnInitialize(handler func(res Resource, initParam any) (resInstance any, err error)) ResourceManager {
	rm.onInitialize = handler
	return rm
}

func (rm *resourceManager) OnRecover(handler func(Resource) error) ResourceManager {
	rm.onRecover = handler
	return rm
}

func (rm *resourceManager) OnSuspend(handler func(Resource) error) ResourceManager {
	rm.onSuspend = handler
	return rm
}

func (rm *resourceManager) OnRelease(handler func(Resource) error) ResourceManager {
	rm.onRelease = handler
	return rm
}

func (r *resource) Put(key string, value any) any {
	r.Lock()
	defer r.Unlock()
	r.ctx[key] = value
	return value
}

func (r *resource) Fetch(key string) (value any, exist bool) {
	r.Unlock()
	defer r.Unlock()
	value, exist = r.ctx[key]
	return
}
