package nbpy

import (
    _ "github.com/packing/nbpy/bits"
    _ "github.com/packing/nbpy/caches"
    _ "github.com/packing/nbpy/codecs"
    _ "github.com/packing/nbpy/containers"
    _ "github.com/packing/nbpy/env"
    _ "github.com/packing/nbpy/errors"
    _ "github.com/packing/nbpy/messages"
    _ "github.com/packing/nbpy/modules"
    _ "github.com/packing/nbpy/net"
    _ "github.com/packing/nbpy/packets"
    _ "github.com/packing/nbpy/utils"
)

func Version() string {
    return "1.135"
}