package utils

import "encoding/base64"

func Base64Encode(src []byte) string {
    return base64.StdEncoding.EncodeToString(src)
}

func Base64EncodeString(src string) string {
    return Base64Encode([]byte(src))
}

func Base64Decode(src string) ([]byte, error) {
    return base64.StdEncoding.DecodeString(src)
}

func Base64DecodeString(src string) (string, error) {
    var r, e = Base64Decode(src)
    if e != nil {
        return "", e
    }
    return string(r), nil
}
