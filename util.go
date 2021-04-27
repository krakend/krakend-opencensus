package opencensus

import (
	"go.opencensus.io/tag"
)

func appendIfMissing(slice []tag.Key, i tag.Key) []tag.Key {
    for _, ele := range slice {
        if ele == i {
            return slice
        }
    }
    return append(slice, i)
}