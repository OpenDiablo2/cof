package cof

// FrameEvent represents a single frame of animation.
type FrameEvent int

// FrameEvent types
const (
	EventNone FrameEvent = iota
	EventAttack
	EventMissile
	EventSound
	EventSkill
)
