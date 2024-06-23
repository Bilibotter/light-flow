package light_flow

type queue[T any] interface {
	Enqueue(T) bool
	Dequeue() (T, bool)
	Len() int
}

type eventHandler struct {
}
