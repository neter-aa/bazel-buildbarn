package circular

// Cursors is a pair of offsets within the data file, indicating which
// part of the file contains valid readable data and where future writes
// need to take place.
type Cursors struct {
	Read  uint64
	Write uint64
}

// Allocate a contiguous range of data.
func (c *Cursors) Allocate(length int64, dataSize uint64) uint64 {
	if length < 1 {
		length = 1
	}
	offset := c.Write
	c.Write += uint64(length)
	if c.Read > c.Write {
		// Overflow of the write counter. Reset.
		c.Read = c.Write
	} else if c.Read+dataSize < c.Write {
		// Invalidate data that is about to be overwritten.
		c.Read = c.Write - dataSize
	}
	return offset
}

// Contains returns whether the provided offset and length are contained
// with the cursors. In effect, it returns whether the offset/length
// pair still corresponds to valid data.
func (c *Cursors) Contains(offset uint64, length int64) bool {
	if length < 1 {
		length = 1
	}
	return offset >= c.Read && offset <= c.Write && offset+uint64(length) <= c.Write
}

// Invalidate all data up to and including the region that is provided.
// This function is called when data corruption is detected.
func (c *Cursors) Invalidate(offset uint64, length int64) {
	if length < 1 {
		length = 1
	}
	c.Read = offset + uint64(length)
	if c.Write < c.Read {
		c.Write = c.Read
	}
}
