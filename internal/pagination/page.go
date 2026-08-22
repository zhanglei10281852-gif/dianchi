package pagination

type Query struct {
	Limit, Offset    int
	State, Chemistry string
}
type Result[T any] struct {
	Items         []T
	Total         int
	Limit, Offset int
}

func Normalize(q Query) Query {
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 25
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	return q
}
