package sharding_test

import (
	"testing"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore/sharding"
	"github.com/EdSchouten/bazel-buildbarn/pkg/mock"
	"github.com/golang/mock/gomock"
)

func TestWeightedShardPermuter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Sequence of backends that should be tried for hash. Every
	// index occurs with the weight provided upon creation.
	shardSelector := mock.NewMockShardSelector(ctrl)
	gomock.InOrder(
		shardSelector.EXPECT().Call(1).Return(true),
		shardSelector.EXPECT().Call(3).Return(true),
		shardSelector.EXPECT().Call(3).Return(true),
		shardSelector.EXPECT().Call(3).Return(true),
		shardSelector.EXPECT().Call(2).Return(true),
		shardSelector.EXPECT().Call(1).Return(true),
		shardSelector.EXPECT().Call(3).Return(true),
		shardSelector.EXPECT().Call(0).Return(true),
		shardSelector.EXPECT().Call(3).Return(true),
		shardSelector.EXPECT().Call(2).Return(true),
		shardSelector.EXPECT().Call(4).Return(true),
		shardSelector.EXPECT().Call(1).Return(true),
		shardSelector.EXPECT().Call(4).Return(true),
		shardSelector.EXPECT().Call(1).Return(true),
		shardSelector.EXPECT().Call(4).Return(false),
	)
	s := sharding.NewWeightedShardPermuter([]uint32{1, 4, 2, 5, 3})
	s.GetShard(9127725482751685232, shardSelector.Call)
}
