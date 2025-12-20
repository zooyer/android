package conf

type Action string

const (
	ActionPress    Action = "PRESS"
	ActionRelease  Action = "RELEASE"
	ActionClick    Action = "CLICK"
	ActionDouble   Action = "DOUBLE"
	ActionLong     Action = "LONG"
	ActionRepeat   Action = "REPEAT"
	ActionCombo    Action = "COMBO"
	ActionPassword Action = "PASSWORD"
)
