// Code generated by "callbackgen -type VIDYA"; DO NOT EDIT.

package indicator

import ()

func (inc *VIDYA) OnUpdate(cb func(value float64)) {
	inc.updateCallbacks = append(inc.updateCallbacks, cb)
}

func (inc *VIDYA) EmitUpdate(value float64) {
	for _, cb := range inc.updateCallbacks {
		cb(value)
	}
}
