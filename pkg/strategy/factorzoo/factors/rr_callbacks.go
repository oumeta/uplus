// Code generated by "callbackgen -type RR"; DO NOT EDIT.

package factorzoo

import ()

func (inc *RR) OnUpdate(cb func(value float64)) {
	inc.updateCallbacks = append(inc.updateCallbacks, cb)
}

func (inc *RR) EmitUpdate(value float64) {
	for _, cb := range inc.updateCallbacks {
		cb(value)
	}
}
