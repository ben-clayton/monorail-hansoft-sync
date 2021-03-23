// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hansoft

import "C"

import (
	"unsafe"
)

type processCallbackHandle = uintptr

var (
	processCallbackHandlers = map[processCallbackHandle]processCallbackHandler{}
	processCallbackHandles  = map[processCallbackHandler]processCallbackHandle{}
)

func registerProcessCallbackHandler(handler processCallbackHandler) processCallbackHandle {
	handle := processCallbackHandle(len(processCallbackHandlers))
	processCallbackHandlers[handle] = handler
	processCallbackHandles[handler] = handle
	return handle
}

func unregisterProcessCallbackHandler(handler processCallbackHandler) {
	handle := processCallbackHandles[handler]
	delete(processCallbackHandlers, handle)
	delete(processCallbackHandles, handler)
}

//export onProcessCallback
func onProcessCallback(handle unsafe.Pointer) {
	handler := processCallbackHandlers[uintptr(handle)]
	handler.onProcessCallback()
}
