package consul_instance_manager

//go:generate go-enum

// ENUM(dead, pending, alive)
type InstanceStatus uint8
