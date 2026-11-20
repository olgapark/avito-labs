package domain

import "errors"

var ErrTripNotFound = errors.New("trip not found")

var ErrDriverBusy = errors.New("driver busy")

var ErrTripCompleted = errors.New("trip completed")
