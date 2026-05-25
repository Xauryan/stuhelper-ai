package common

import (
	"io"

	"github.com/Xauryan/stuhelper-ai/common"
)

// NewOutboundJSONBody wraps an already-marshaled upstream request body in
// BodyStorage. Disk cache mode can then move large payloads, such as base64
// image requests, out of heap while waiting for the upstream response.
//
// The caller must close closer after the upstream call finishes.
func NewOutboundJSONBody(data []byte) (body io.Reader, size int64, closer io.Closer, err error) {
	storage, err := common.CreateBodyStorage(data)
	if err != nil {
		return nil, 0, nil, err
	}
	return common.ReaderOnly(storage), storage.Size(), storage, nil
}
