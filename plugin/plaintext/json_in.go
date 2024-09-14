package plaintext

import (
	"encoding/json"

	"geoip/lib"
)

const (
	TypeJSONIn = "json"
	DescJSONIn = "Convert JSON data to other formats"
)

func init() {
	lib.RegisterInputConfigCreator(TypeJSONIn, func(action lib.Action, data json.RawMessage) (lib.InputConverter, error) {
		return newTextIn(TypeJSONIn, DescJSONIn, action, data)
	})

	lib.RegisterInputConverter(TypeJSONIn, &TextIn{
		Description: DescJSONIn,
	})
}
