package light_flow

import (
	"sync"
)

const (
	initializeR = "initialize"
	releaseR    = "release"
	suspendR    = "suspend"
	recoverR    = "recover"
	attachR     = "attach"
)

var resourceManagers = make(map[string]*resourceManager)
var resourceLock = new(sync.RWMutex)

type Resource interface {
	nameI
	ProcessName() string
	ProcessID() string
	Entity() any
	Put(key string, value any) any
	Fetch(key string) (value any, exist bool)
	Update(entity any) any
	Clear() any
}

type boundProc interface {
	nameI
	identifierI
	sync.Locker
}

type ResourceManager interface {
	OnInitialize(func(res Resource, initParam any) (entity any, err error)) ResourceManager
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
	boundProc
	resName string
	ctx     map[string]any
	entity  any
}

type resSerializable struct {
	Name   string
	Ctx    map[string]any
	Entity any
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
		name:         name,
		onInitialize: emptyInitialize,
		onRecover:    emptyHandler,
		onSuspend:    emptyHandler,
		onRelease:    emptyHandler,
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

func emptyInitialize(_ Resource, _ any) (entity any, err error) {
	return nil, nil
}

func (rm *resourceManager) OnInitialize(handler func(res Resource, initParam any) (entity any, err error)) ResourceManager {
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

func (r *resource) Name() string {
	return r.resName
}

func (r *resource) ProcessName() string {
	return r.boundProc.Name()
}

func (r *resource) ProcessID() string {
	return r.boundProc.ID()
}

func (r *resource) Entity() any {
	return r.entity
}

func (r *resource) Put(key string, value any) any {
	r.Lock()
	defer r.Unlock()
	if r.ctx == nil {
		r.ctx = make(map[string]any, 1)
	}
	r.ctx[key] = value
	return value
}

func (r *resource) Fetch(key string) (value any, exist bool) {
	r.Lock()
	defer r.Unlock()
	value, exist = r.ctx[key]
	return
}

func (r *resource) Update(entity any) any {
	r.Lock()
	defer r.Unlock()
	r.entity = entity
	return entity
}

func (r *resource) Clear() any {
	r.Lock()
	defer r.Unlock()
	res := r.entity
	r.ctx = nil
	r.entity = nil
	return res
}
