package machine

import policybinding "github.com/Kinirin/PTSIP/developer/automation/policy/binding"

const BindingRegistryPath = policybinding.BindingRegistryPath
const BindingSchemaPath = policybinding.BindingSchemaPath

type BindingSnapshot = policybinding.BindingSnapshot

func (r *Repository) LoadBindingRegistry(required bool) (*BindingSnapshot, error) {
	return policybinding.NewStore(r).LoadBindingRegistry(required)
}
func (r *Repository) ResolveBindings(query Object) (Object, error) {
	return policybinding.NewStore(r).ResolveBindings(query)
}
func (r *Repository) ValidateBindingRegistry(payload Object) []string {
	return policybinding.NewStore(r).ValidateBindingRegistry(payload)
}
func (r *Repository) ReplaceBindingRegistry(payload Object, digest string, validate func(Object) []string) (*BindingSnapshot, error) {
	return policybinding.NewStore(r).ReplaceBindingRegistry(payload, digest, validate)
}
func (r *Repository) ReconcileBindings(apply bool) (Object, error) {
	return policybinding.NewStore(r).ReconcileBindings(apply)
}
func planningAtomicWrite(path string, raw []byte) error {
	return policybinding.WriteFileAtomically(path, raw)
}
