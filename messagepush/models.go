package messagepush

type PushMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}
