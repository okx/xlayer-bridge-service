package messagepush

type PushMessage[T any] struct {
	Type string `json:"type"`
	Data []T    `json:"data"`
}
