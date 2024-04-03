package nodemanager

import (
	"context"
	"fmt"
	nodev1 "github.com/piwriw/nodedeploy-controller/api/v1"
	utils_ssh "github.com/piwriw/nodedeploy-controller/utils/ssh"
)

type NodeDisJoin interface {
	DisJoin(ctx context.Context, sshclient *utils_ssh.Client) error
}

func (wn WorkNode) DisJoin(ctx context.Context, sshclient *utils_ssh.Client) error {

	workerNode_disjoin_cmd := fmt.Sprintf("bash %s/%s/08-work-disjoin.sh ", FilePathPrefix, nodev1.NodeWork.String())
	_, err := sshclient.Exec(ctx, workerNode_disjoin_cmd)
	if err != nil {
		return fmt.Errorf("%s, err: %w", "fail to deprecate", err)
	}
	return nil
}
