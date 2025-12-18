package domain

type ChannelLifecycleState string

const (
	ChannelStateActive   ChannelLifecycleState = "active"
	ChannelStateInactive ChannelLifecycleState = "inactive"
	ChannelStateDelete   ChannelLifecycleState = "delete"
)

type ChannelControl struct {
	State ChannelLifecycleState `json:"state"`
}
