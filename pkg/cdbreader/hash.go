package cdbreader

func hash(key []byte) uint32 {
	h := uint32(5381)
	for _, c := range key {
		h = (h + (h << 5)) ^ uint32(c)
	}
	return h
}
