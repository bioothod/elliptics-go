/*
 * 2013+ Copyright (c) Anton Tyurin <noxiouz@yandex.ru>
 * 2014+ Copyright (c) Evgeniy Polyakov <zbr@ioremap.net>
 * All rights reserved.
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU General Public License for more details.
 */

package elliptics

/*
#include "session.h"
#include <stdio.h>

struct dnet_iterator_response_unpacked {
        uint64_t                        id;
        struct dnet_raw_id              key;
        int                             status;
        struct dnet_time                timestamp;
        uint64_t                        user_flags;
        uint64_t                        size;
        uint64_t                        iterated_keys;
        uint64_t                        total_keys;
        uint64_t                        flags;
};

static inline void unpack_dnet_iterator_response(struct dnet_iterator_response *from,
	 struct dnet_iterator_response_unpacked *to)
{
	to->id = from->id;
    to->key = from->key;
    to->status = from->status;
    to->timestamp = from->timestamp;
    to->user_flags = from->user_flags;
    to->size = from->size;
    to->iterated_keys = from->iterated_keys;
    to->total_keys = from->total_keys;
    to->flags = from->flags;
}

*/
import "C"

import (
	"time"
	"unsafe"
)

type DnetIteratorResponse struct {
	Id           uint64
	Key          C.struct_dnet_raw_id
	Status       int
	Timestamp    time.Time
	UserFlags    uint64
	Size         uint64
	IteratedKeys uint64
	TotalKeys    uint64
	Flags        uint64
}

type IteratorResult interface {
	Reply() *DnetIteratorResponse
	ReplyData() []byte
	Id() uint64
	Error() error
}

type iteratorResult struct {
	reply     *DnetIteratorResponse
	replyData []byte
	id        uint64
	err       error
}

func (i *iteratorResult) Reply() *DnetIteratorResponse { return i.reply }

func (i *iteratorResult) ReplyData() []byte { return i.replyData }

func (i *iteratorResult) Id() uint64 { return i.id }

func (i *iteratorResult) Error() error { return i.err }

//export go_iterator_callback
func go_iterator_callback(result *C.struct_go_iterator_result, key uint64) {
	context, err := Pool.Get(key)
	if err != nil {
		panic("Unable to find session numbder")
	}

	callback := context.(func(*iteratorResult))

	var reply C.struct_dnet_iterator_response_unpacked

	C.unpack_dnet_iterator_response(result.reply, &reply)

	var Result = iteratorResult{
		reply: &DnetIteratorResponse{
			Id:           uint64(result.reply.id),
			Key:          reply.key,
			Status:       int(reply.status),
			Timestamp:    time.Unix(int64(reply.timestamp.tsec), int64(reply.timestamp.tnsec)),
			UserFlags:    uint64(reply.user_flags),
			Size:         uint64(reply.size),
			IteratedKeys: uint64(reply.iterated_keys),
			TotalKeys:    uint64(reply.total_keys),
			Flags:        uint64(reply.flags),
		},
		id:        uint64(result.id),
		replyData: nil,
	}

	if result.reply_size > 0 && result.reply_data != nil {
		Result.replyData = C.GoBytes(unsafe.Pointer(result.reply_data), C.int(result.reply_size))
	} else {
		Result.replyData = make([]byte, 0)
	}

	callback(&Result)
}

func iteratorHelper(key string, iteratorId uint64) (*Key, uint64, uint64, <-chan IteratorResult) {
	ekey, err := NewKey(key)
	if err != nil {
		panic(err)
	}

	responseCh := make(chan IteratorResult, defaultVOLUME)

	onResultContext := NextContext()
	onFinishContext := NextContext()

	onResult := func(iterres *iteratorResult) {
		responseCh <- iterres
	}

	onFinish := func(err error) {
		if err != nil {
			responseCh <- &iteratorResult{err: err}
		}
		close(responseCh)

		Pool.Delete(onResultContext)
		Pool.Delete(onFinishContext)
	}

	Pool.Store(onResultContext, onResult)
	Pool.Store(onFinishContext, onFinish)
	return ekey, onResultContext, onFinishContext, responseCh
}

func (s *Session) IteratorPause(key string, iteratorId uint64) <-chan IteratorResult {
	ekey, onResultContext, onFinishContext, responseCh := iteratorHelper(key, iteratorId)
	defer ekey.Free()

	C.session_pause_iterator(s.session, C.context_t(onResultContext), C.context_t(onFinishContext),
		ekey.key,
		C.uint64_t(iteratorId))
	return responseCh
}

func (s *Session) IteratorContinue(key string, iteratorId uint64) <-chan IteratorResult {
	ekey, onResultContext, onFinishContext, responseCh := iteratorHelper(key, iteratorId)
	defer ekey.Free()

	C.session_continue_iterator(s.session, C.context_t(onResultContext), C.context_t(onFinishContext),
		ekey.key,
		C.uint64_t(iteratorId))
	return responseCh
}

func (s *Session) IteratorStop(key string, iteratorId uint64) <-chan IteratorResult {
	ekey, onResultContext, onFinishContext, responseCh := iteratorHelper(key, iteratorId)
	defer ekey.Free()

	C.session_stop_iterator(s.session, C.context_t(onResultContext), C.context_t(onFinishContext),
		ekey.key,
		C.uint64_t(iteratorId))
	return responseCh
}
