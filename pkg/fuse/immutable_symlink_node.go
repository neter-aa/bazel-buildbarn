package fuse

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type immutableSymlinkNode struct {
	nodefs.Node
	target    string
}

// NewImmutableSymlinkNode creates a FUSE symlink node.
func NewImmutableSymlinkNode(target string) nodefs.Node {
	return &immutableSymlinkNode{
		Node:      nodefs.NewDefaultNode(),
		target:    target,
	}
}

func (n *immutableSymlinkNode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
	*out = fuse.Attr{
		Size: uint64(len(n.target)),
		Mode: fuse.S_IFLNK | 0777,
	}
	return fuse.OK
}

func (n *immutableSymlinkNode) Readlink(c *fuse.Context) ([]byte, fuse.Status) {
	return []byte(n.target), fuse.OK
}
