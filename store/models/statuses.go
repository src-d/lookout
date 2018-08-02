package models

type EventStatus string

const (
	EventStatusNew       = EventStatus("new")
	EventStatusProcessed = EventStatus("processed")
	EventStatusFailed    = EventStatus("failed")
)
