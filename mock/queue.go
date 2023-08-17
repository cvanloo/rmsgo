package mock

type Queue[T any] []T

func (q *Queue[T]) Dequeue() T {
	el := (*q)[0]
	*q = (*q)[1:]
	return el
}

func (q *Queue[T]) Empty() bool {
	return len(*q) == 0
}
