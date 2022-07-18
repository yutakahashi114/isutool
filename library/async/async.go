package async

func Go[T any](fn func() (T, error)) Result[T] {
	ch := make(chan result[T], 1)
	go func() {
		res, err := fn()
		ch <- result[T]{
			res: res,
			err: err,
		}
		close(ch)
	}()
	return Result[T]{ch: ch}
}

type Result[T any] struct {
	ch chan result[T]
}

func (r *Result[T]) Await() (T, error) {
	result := <-r.ch
	return result.res, result.err
}

type result[T any] struct {
	res T
	err error
}
