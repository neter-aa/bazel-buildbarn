package sharding_test

import (
	"testing"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore/sharding"
	"github.com/EdSchouten/bazel-buildbarn/pkg/mock"
	"github.com/golang/mock/gomock"
)

func TestWeightedShardSelector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Sequence of backends that should be tried for hash. Every
	// index occurs with the weight provided upon creation.
	shardProposer := mock.NewMockShardProposer(ctrl)
	gomock.InOrder(
		shardProposer.EXPECT().Call(1).Return(true),
		shardProposer.EXPECT().Call(3).Return(true),
		shardProposer.EXPECT().Call(3).Return(true),
		shardProposer.EXPECT().Call(3).Return(true),
		shardProposer.EXPECT().Call(2).Return(true),
		shardProposer.EXPECT().Call(1).Return(true),
		shardProposer.EXPECT().Call(3).Return(true),
		shardProposer.EXPECT().Call(0).Return(true),
		shardProposer.EXPECT().Call(3).Return(true),
		shardProposer.EXPECT().Call(2).Return(true),
		shardProposer.EXPECT().Call(4).Return(true),
		shardProposer.EXPECT().Call(1).Return(true),
		shardProposer.EXPECT().Call(4).Return(true),
		shardProposer.EXPECT().Call(1).Return(true),
		shardProposer.EXPECT().Call(4).Return(false),
	)
	s := sharding.NewWeightedShardSelector([]uint32{1, 4, 2, 5, 3})
	s.GetShard(9127725482751685232, shardProposer.Call)
}
