package ovirtclient

import (
	"fmt"

	ovirtsdk "github.com/ovirt/go-ovirt"
)

func (o *oVirtClient) UpdateVM(
	id VMID,
	params UpdateVMParameters,
	retries ...RetryStrategy,
) (result VM, err error) {
	retries = defaultRetries(retries, defaultWriteTimeouts(o))
	o.logger.Infof("updating VM params %s", params)
	vm := &ovirtsdk.Vm{}
	vm_cores := &ovirtsdk.CpuTopology{}
	vm_cpu := &ovirtsdk.Cpu{}
	vm.SetId(string(id))
	if name := params.Name(); name != nil {
		if *name == "" {
			return nil, newError(EBadArgument, "name must not be empty for VM update")
		}
		vm.SetName(*name)
	}
	if comment := params.Comment(); comment != nil {
		vm.SetComment(*comment)
	}
	if description := params.Description(); description != nil {
		vm.SetDescription(*description)
	}
	if memory := params.Memory(); memory != nil {
		vm.SetMemory(int64(*memory))
	}
	vm_cores.SetCores(int64(*params.CpuCores()))
	vm_cores.SetThreads(int64(*params.CpuThreads()))
	vm_cores.SetSockets(int64(*params.CpuSockets()))
	vm_cpu.SetTopology(vm_cores)
	vm.SetCpu(vm_cpu)

	err = retry(
		fmt.Sprintf("updating vm %s", id),
		o.logger,
		retries,
		func() error {
			response, err := o.conn.SystemService().VmsService().VmService(string(id)).Update().Vm(vm).Send()
			if err != nil {
				return wrap(err, EUnidentified, "failed to update VM")
			}
			vm, ok := response.Vm()
			if !ok {
				return newError(EFieldMissing, "missing VM in VM update response")
			}
			result, err = convertSDKVM(vm, o)
			if err != nil {
				return wrap(
					err,
					EBug,
					"failed to convert VM",
				)
			}

			return nil
		})
	return result, err
}

func (m *mockClient) UpdateVM(id VMID, params UpdateVMParameters, _ ...RetryStrategy) (VM, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.vms[id]; !ok {
		return nil, newError(ENotFound, "VM with ID %s not found", id)
	}

	vm := m.vms[id]
	if name := params.Name(); name != nil {
		for _, otherVM := range m.vms {
			if otherVM.name == *name && otherVM.ID() != vm.ID() {
				return nil, newError(EConflict, "A VM with the name \"%s\" already exists.", *name)
			}
		}
		vm = vm.withName(*name)
	}
	if comment := params.Comment(); comment != nil {
		vm = vm.withComment(*comment)
	}
	if description := params.Description(); description != nil {
		vm = vm.withDescription(*description)
	}
	m.vms[id] = vm

	return vm, nil
}
