package xmlui

import (
	"fmt"
	"github.com/kellydornhaus/layouter/layout"
	"reflect"
)

// BindByID assigns components from r.ByID into fields of dest.
// Fields that can be set:
// - Tagged fields: `ui:"id"`
// - Untagged fields whose name matches an id (case-sensitive)
// Field types can be layout.Component or a concrete component pointer
// (e.g., *layout.PanelComponent). A type mismatch skips the field.
func BindByID(dest any, r Result) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("dest must be non-nil pointer")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("dest must point to a struct")
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		sf := v.Type().Field(i)
		id := sf.Tag.Get("ui")
		if id == "" {
			id = sf.Name
		}
		comp, ok := r.ByID[id]
		if !ok {
			continue
		}
		// assign if types match or field is layout.Component
		if f.Type() == reflect.TypeOf((*layout.Component)(nil)).Elem() {
			f.Set(reflect.ValueOf(comp))
			continue
		}
		cv := reflect.ValueOf(comp)
		if cv.Type().AssignableTo(f.Type()) {
			f.Set(cv)
		}
	}
	return nil
}
