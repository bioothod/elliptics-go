package elliptics

/*
#include "session.h"
*/
import "C"

import (
	"fmt"
	"reflect"
	"unsafe"
)

var _ = fmt.Println

//export go_final_callback
func go_final_callback(err int, context unsafe.Pointer) {
	callback := *(*func(int))(context)
	callback(err)
}

//export go_lookup_callback
func go_lookup_callback(result *C.struct_go_lookup_result, context unsafe.Pointer) {
	callback := *(*func(*lookupResult))(context)
	Result := lookupResult{
		info: *result.info,
		addr: *result.addr,
		path: C.GoString(result.path),
		err:  nil,
	}
	callback(&Result)
}

//export go_read_callback
func go_read_callback(item *C.struct_go_read_result, context unsafe.Pointer) {
	callback := *(*func(readResult))(context)

	Result := readResult{
		data:   C.GoStringN(item.file, C.int(item.size)),
		ioAttr: *item.io_attribute,
		err:    nil,
	}
	// All data from C++ has been copied here.
	callback(Result)
}

//export go_remove_callback
// func go_remove_callback(err int, context unsafe.Pointer) {
// 	callback := *(*func(int))(context)
// 	callback(err)
// }

//export go_find_callback
func go_find_callback(result *C.struct_go_find_result, context unsafe.Pointer) {
	callback := *(*func(*findResult))(context)
	var indexEntries []C.struct_c_index_entry
	size := int(result.entries_count)
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&indexEntries)))
	sliceHeader.Cap = size
	sliceHeader.Len = size
	sliceHeader.Data = uintptr(unsafe.Pointer(result.entries))
	var IndexDatas []IndexEntry
	for _, item := range indexEntries {
		IndexDatas = append(IndexDatas, IndexEntry{
			Data: C.GoStringN(item.data, C.int(item.size)),
		})
	}
	callback(&findResult{
		id:   *result.id,
		data: IndexDatas,
		err:  nil,
	})
}
