package utils

import "encoding/json"

func MarshalString(v any) string {
	data, _ := json.Marshal(v)

	return string(data)
}
