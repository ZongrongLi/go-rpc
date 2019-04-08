package protocol

import (
	"encoding/json"
	"errors"
)

//新加入的序列化器 必须在这里定义
var serializerMap = map[SerializeType]Serializer{
	SerializeTypeJson: JsonSerializer{},
}

type Serializer interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

type JsonSerializer struct {
}

func (s JsonSerializer) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)

}

func (s JsonSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func NewSerializer(typ SerializeType) (Serializer, error) {
	s, ok := serializerMap[typ]
	if !ok {
		return nil, errors.New("Serializer not exist")
	}
	return s, nil
}
