package codecs

type DecoderJSONv1 struct {
}

type EncoderJSONv1 struct {
}


func (receiver DecoderJSONv1) Decode(raw []byte) (error, IMData, []byte){
	return nil, nil, raw
}

func (receiver EncoderJSONv1) Encode(raw *IMData) (error, []byte){
	return nil, []byte("")
}

var codecJSONv1 = Codec{Protocol:ProtocolJSON, Version:1, Decoder: DecoderJSONv1{}, Encoder: EncoderJSONv1{}}
var CodecJSONv1 = &codecJSONv1