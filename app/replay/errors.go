package replay

import (
	"errors"
)

var ErrAuthServerDown = errors.New("authentication server did not respond")
var ErrAuthUnauthorized = errors.New("unauthorized")
