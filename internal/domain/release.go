package domain

import "time"

type Release struct {
	AppName       string
	Version       string
	Status        string
	StoragePrefix string
	CommitSHA     string
	BuildID       string
	CreatedAt     time.Time
}
