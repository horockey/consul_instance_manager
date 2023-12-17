package go-consul-instance-manager

type Instance struct {
	name    string
	address string
	status  InstanceStatus
}

func (ins *Instance) Name() string {
	return ins.name
}

func (ins *Instance) Address() string {
	return ins.address
}

func (ins *Instance) Status() InstanceStatus {
	return ins.status
}
