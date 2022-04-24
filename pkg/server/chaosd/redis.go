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

package chaosd

import (
	"fmt"
	"math"
	"os/exec"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pingcap/errors"

	"github.com/chaos-mesh/chaosd/pkg/core"
)

type redisAttack struct{}

var RedisAttack AttackType = redisAttack{}

const (
	STATUSOK = "OK"
)

func (redisAttack) Attack(options core.AttackConfig, env Environment) error {
	attack := options.(*core.RedisCommand)

	cli := redis.NewClient(&redis.Options{
		Addr:     attack.Addr,
		Password: attack.Password,
	})
	_, err := cli.Ping(cli.Context()).Result()
	if err != nil {
		return errors.WithStack(err)
	}

	switch attack.Action {
	case core.RedisSentinelRestartAction:
		err := env.Chaos.shutdownSentinelServer(attack, cli)
		if err != nil {
			return errors.WithStack(err)
		}
		return env.Chaos.recoverSentinelStop(attack)

	case core.RedisSentinelStopAction:
		return env.Chaos.shutdownSentinelServer(attack, cli)

	case core.RedisCachePenetrationAction:
		var wg sync.WaitGroup
		start := time.Now()
		num := int(math.Ceil(float64(attack.Frequency) / float64(attack.ConcurrentNum)))
		concurrentNum := attack.ConcurrentNum
		wg.Add(attack.Frequency)
		for i := 0; i < num; i++ {
			if attack.Frequency-i*concurrentNum < concurrentNum {
				concurrentNum = attack.Frequency - i*concurrentNum
			}

			go func(concurrentNum int) {
				for j := 0; j < concurrentNum; j++ {
					cli.Get(cli.Context(), "CHAOS_MESH_nqE3BWm7khHv")
					wg.Done()
				}
			}(concurrentNum)
		}
		// limitedChannel := make(chan struct{}, attack.ConcurrentNum)
		// wg.Add(attack.Frequency)
		// for i := 0; i < attack.Frequency; i++ {
		// 	limitedChannel <- struct{}{}

		// 	go func() {
		// 		defer wg.Done()
		// 		cli.Get(cli.Context(), "CHAOS_MESH_nqE3BWm7khHv")
		// 		<-limitedChannel
		// 	}()
		// }
		wg.Wait()
		fmt.Println(time.Now().Sub(start))
	}
	return nil
}

func (redisAttack) Recover(exp core.Experiment, env Environment) error {
	config, err := exp.GetRequestCommand()
	if err != nil {
		return err
	}
	attack := config.(*core.RedisCommand)

	switch attack.Action {
	case core.RedisSentinelStopAction:
		return env.Chaos.recoverSentinelStop(attack)
	}
	return nil
}

func (s *Server) shutdownSentinelServer(attack *core.RedisCommand, cli *redis.Client) error {
	if attack.FlushConfig {
		// Because redis.Client doesn't have the func `FlushConfig()`, a redis.SentinelClient has to be created
		sentinelCli := redis.NewSentinelClient(&redis.Options{
			Addr: attack.Addr,
		})
		result, err := sentinelCli.FlushConfig(sentinelCli.Context()).Result()
		if err != nil {
			return errors.WithStack(err)
		}
		if result != STATUSOK {
			return errors.WithStack(errors.Errorf("redis command status is %s", result))
		}
	}

	// If cli.Shutdown() runs successfully, the result will be nil and the err will be "connection refused"
	result, err := cli.Shutdown(cli.Context()).Result()
	if result != "" {
		return errors.WithStack(err)
	}

	return nil
}

func (s *Server) recoverSentinelStop(attack *core.RedisCommand) error {
	if attack.Conf == "" {
		return errors.WithStack(errors.Errorf("redis config is not present"))
	}
	recoverCmd := exec.Command("redis-server", attack.Conf, "--sentinel")
	_, err := recoverCmd.CombinedOutput()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
