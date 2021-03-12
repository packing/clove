package errors

var ErrorAddress = Errorf("The address is error")
var ErrorDataSentIncomplete = Errorf("The data sent is incomplete")
var ErrorPacketFormatNotReady = Errorf("The packet format is not ready")
var ErrorCodecNotReady = Errorf("The codec is not ready")
var ErrorDecryptFunctionNotBind = Errorf("The packet is encrypted, but the decrypt function is not bind")
var ErrorUncompressFunctionNotBind = Errorf("The packet is compressed, but the uncompress function is not bind")

var ErrorSessionIsNotExists = Errorf("The session is not exists")

var ErrorDataNotEnough = Errorf("The length of the head data is not enough to be decoded")
var ErrorDataTooShort = Errorf("The length of the head data is too short")
var ErrorTypeNotSupported = Errorf("Type is not supported")

var ErrorDataNotReady = Errorf("Data length is not enough")
var ErrorDataNotMatch = Errorf("Cannot match any packet format")
var ErrorDataIsDamage = Errorf("Data length is not match")
var ErrorRemoteReqClose = Errorf("The remote host request close it")
