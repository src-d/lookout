package models

type EventStatus string

const (
	EventStatusNew       = EventStatus("new")
	EventStatusPosting   = EventStatus("posting")
	EventStatusProcessed = EventStatus("processed")
	EventStatusFailed    = EventStatus("failed")
)
