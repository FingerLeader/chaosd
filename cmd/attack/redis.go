// Copyright 2022 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package attack

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/chaos-mesh/chaosd/cmd/server"
	"github.com/chaos-mesh/chaosd/pkg/core"
	"github.com/chaos-mesh/chaosd/pkg/server/chaosd"
	"github.com/chaos-mesh/chaosd/pkg/utils"
)

func NewRedisAttackCommand(uid *string) *cobra.Command {
	options := core.NewRedisCommand()
	dep := fx.Options(
		server.Module,
		fx.Provide(func() *core.RedisCommand {
			options.UID = *uid
			return options
		}),
	)

	cmd := &cobra.Command{
		Use:   "redis <subcommand>",
		Short: "Redis attack related commands",
	}

	cmd.AddCommand(
		// NewRedisSentinelRestartCommand(dep, options),
		NewRedisSentinelStopCommand(dep, options),
	)

	return cmd
}

func NewRedisSentinelRestartCommand(dep fx.Option, options *core.RedisCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "restart sentinel",
		Run: func(*cobra.Command, []string) {
			options.Action = core.RedisSentinelRestartAction
			utils.FxNewAppWithoutLog(dep, fx.Invoke(redisAttackF)).Run()
		},
	}

	return cmd
}

func NewRedisSentinelStopCommand(dep fx.Option, options *core.RedisCommand) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "sentinel-stop",
		Short: "stop sentinel",
		Run: func(*cobra.Command, []string) {
			options.Action = core.RedisSentinelStopAction
			utils.FxNewAppWithoutLog(dep, fx.Invoke(redisAttackF)).Run()
		},
	}

	cmd.Flags().StringVarP(&options.Addr, "addr", "a", "", "")
	cmd.Flags().StringVarP(&options.Password, "password", "p", "", "The signal")
	cmd.Flags().IntVarP(&options.DB, "db", "", 0, "The command to be run when recovering experiment")
	cmd.Flags().StringVarP(&options.Conf, "conf", "c", "", "The config of redis server")

	return cmd
}

func redisAttackF(chaos *chaosd.Server, options *core.RedisCommand) {
	if err := options.Validate(); err != nil {
		utils.ExitWithError(utils.ExitBadArgs, err)
	}
	uid, err := chaos.ExecuteAttack(chaosd.RedisAttack, options, core.CommandMode)
	if err != nil {
		utils.ExitWithError(utils.ExitError, err)
	}

	utils.NormalExit(fmt.Sprintf("Attack redis %s successfully, uid: %s", options.Action, uid))
}
