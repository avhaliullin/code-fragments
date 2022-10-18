package main

import (
	"context"
	"fmt"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"google.golang.org/genproto/protobuf/field_mask"
	"sort"
	"strconv"
	"time"
)

const labelLastRestarted = "last-restarted"

func Handler(ctx context.Context, _ interface{}) (string, error) { //nolint: deadcode,unused
	defer log.Sync()
	err := doHandle(ctx)
	if err != nil {
		log.Error(err.Error())
		return "", err
	}
	return "", nil
}

func doHandle(ctx context.Context) error {
	initConf()

	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: ycsdk.InstanceServiceAccount(),
	})
	if err != nil {
		return err
	}

	vms, err := selectInstancesToUpdate(ctx, sdk)
	if err != nil {
		return err
	}
	for _, vm := range vms {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		switch vm.GetStatus() {
		case compute.Instance_RUNNING:
			log.Info(fmt.Sprintf("stopping instance %s", vm.GetId()))
			_, err = sdk.Compute().Instance().Stop(ctx, &compute.StopInstanceRequest{InstanceId: vm.GetId()})
			if err != nil {
				log.Error(fmt.Sprintf("failed to stop instance %s: %s", vm.GetId(), err))
				continue
			}
		case compute.Instance_STOPPED:
			log.Info(fmt.Sprintf("starting instance %s", vm.GetId()))
			labels := vm.GetLabels()
			labels[labelLastRestarted] = strconv.FormatInt(time.Now().Unix(), 10)
			_, err := sdk.WrapOperation(sdk.Compute().Instance().Update(ctx, &compute.UpdateInstanceRequest{
				InstanceId: vm.GetId(),
				UpdateMask: &field_mask.FieldMask{
					Paths: []string{"labels"},
				},
				Labels: labels,
			}))
			if err != nil {
				log.Error(fmt.Sprintf("failed to update instance %s labels: %s", vm.GetId(), err))
				continue
			}
			_, err = sdk.Compute().Instance().Start(ctx, &compute.StartInstanceRequest{InstanceId: vm.GetId()})
			if err != nil {
				log.Error(fmt.Sprintf("failed to start instance %s labels: %s", vm.GetId(), err))
				continue
			}
		}
	}
	return nil
}
func selectInstancesToUpdate(ctx context.Context, sdk *ycsdk.SDK) ([]*compute.Instance, error) {
	var result []*compute.Instance
	pageToken := ""
	opsInflight := 0

	for {
		resp, err := sdk.Compute().Instance().List(ctx, &compute.ListInstancesRequest{
			FolderId:  conf.folderID,
			PageToken: pageToken,
		})
		if err != nil {
			return nil, err
		}
		for _, vm := range resp.GetInstances() {
			if _, ok := vm.GetLabels()[conf.restartLabel]; !ok {
				continue
			}
			switch vm.GetStatus() {
			case compute.Instance_PROVISIONING:
			case compute.Instance_STARTING:
			case compute.Instance_UPDATING:
			case compute.Instance_RESTARTING:
			case compute.Instance_STOPPING:
			case compute.Instance_DELETING:
				// operations, doing nothing
				opsInflight++
			case compute.Instance_STOPPED:
				// schedule to start
				result = append(result, vm)
			case compute.Instance_RUNNING:
				if shouldRestart(vm) {
					// schedule to stop
					result = append(result, vm)
				}
			case compute.Instance_ERROR:
			case compute.Instance_CRASHED:
			case compute.Instance_STATUS_UNSPECIFIED:
				log.Warn(fmt.Sprintf("unexpected instance %s status: %s, ignoring", vm.GetId(), vm.GetStatus()))
				continue
			}
		}
		if pageToken = resp.GetNextPageToken(); len(pageToken) == 0 {
			break
		}
	}
	sort.Slice(result, func(i, j int) bool {
		st1 := result[i].GetStatus()
		st2 := result[j].GetStatus()
		if st1 != st2 && st1 == compute.Instance_STOPPED {
			// less is higher priority
			// stopped is highest priority
			return true
		}
		return false
	})

	if len(result)+opsInflight > conf.opsLimit {
		log.Info(fmt.Sprintf(
			"postponing %d operations: got %d instances to operate, %d ops inflight and %d ops limit",
			len(result)+opsInflight-conf.opsLimit, len(result), opsInflight, conf.opsLimit,
		))
		toKeep := conf.opsLimit - opsInflight
		if toKeep <= 0 {
			return nil, nil
		}
		result = result[:toKeep]
	}
	return result, nil
}

func shouldRestart(vm *compute.Instance) bool {
	if !conf.inMaintenance {
		return false
	}
	createdAt := time.Unix(vm.GetCreatedAt().GetSeconds(), int64(vm.GetCreatedAt().GetNanos())).UTC()
	untouchedAfter := time.Now().Add(-time.Duration(conf.hoursToRestart) * time.Hour)
	if createdAt.After(untouchedAfter) {
		return false
	}
	if restartedStr, ok := vm.GetLabels()[labelLastRestarted]; ok {
		restartTs, err := strconv.ParseInt(restartedStr, 10, 64)
		if err != nil {
			return true
		}
		restartedAt := time.Unix(restartTs, 0)
		if restartedAt.After(untouchedAfter) {
			return false
		}
	}
	return true
}
