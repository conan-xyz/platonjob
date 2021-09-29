package sched

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/klog"

	"gitee.com/zonzpoo/platonjob/conf"
	"gitee.com/zonzpoo/platonjob/internal"
)

// Controller is schedule controller
type Controller struct {
	ctx    context.Context
	cancel context.CancelFunc

	lock *sync.RWMutex

	svc internal.SvcImpl

	rewardBlock   int64
	delegateBlock int64

	rewardCycle   int64
	delegateCycle int64

	canReward   bool
	canDelegate bool
}

func NewController(parent context.Context, ac *conf.Config) *Controller {
	var (
		err error
	)
	c := &Controller{
		lock: &sync.RWMutex{},
	}
	c.ctx, c.cancel = context.WithCancel(parent)
	c.svc, err = internal.New(c.ctx, ac)
	if err != nil {
		panic(err)
	}

	c.rewardBlock, c.delegateBlock = ac.RewardBlock, ac.DelegateBlock
	c.rewardCycle, c.delegateCycle = c.currentCycle(), c.currentCycle()
	c.canReward, c.canDelegate = false, false

	// default setting
	if c.rewardBlock == 0 {
		c.rewardBlock = 8000
	}
	if c.delegateBlock == 0 {
		c.delegateBlock = 3000
	}

	return c
}

// WithdrawReward ...
func (c *Controller) WithdrawReward() {
	t := time.NewTicker(time.Minute)
	for {
		select {
		case <-c.ctx.Done():
			klog.Info("[WithdrawReward] Received stop signal, exited")
			return
		case <-t.C:
			go c.getReward()
		}
	}
}

func (c *Controller) getReward() (err error) {
	remain := c.remainCycleNumber()
	canDo := c.safeGetRewardCanDo()
	klog.Infof("[getReward] current remain cycle blocknumber %d, diff blocknumber %d", remain, c.rewardBlock)
	if canDo && remain <= c.rewardBlock {
		c.svc.WithdrawReward(c.ctx)
		c.safeAddRewardCycle()
	}
	return
}

func (c *Controller) RunDelegate() (err error) {
	t := time.NewTicker(time.Minute)
	for {
		select {
		case <-c.ctx.Done():
			klog.Info("[RunDelegate] Received stop signal, exited")
			return
		case <-t.C:
			go c.initDelegate()
		}
	}
}

func (c *Controller) initDelegate() (err error) {
	remain := c.remainCycleNumber()
	canDo := c.safeGetDelegateCanDo()
	klog.Infof("[initDelegate] current remain cycle blocknumber %d, diff blocknumber %d", remain, c.delegateBlock)
	if canDo && remain <= c.delegateBlock {
		c.svc.InitDelegate(c.ctx)
		c.safeAddDelegateCycle()
	}
	return
}

func (c *Controller) currentCycle() int64 {
	number := c.svc.CurrentBlockNumber(c.ctx)
	return number/10750 + 1
}

func (c *Controller) remainCycleNumber() int64 {
	number := c.svc.CurrentBlockNumber(c.ctx)
	cycle := c.currentCycle()
	return 10750*cycle - number
}

func (c *Controller) safeSetRewardCanDo(do bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.canReward = do
}

func (c *Controller) safeGetRewardCanDo() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.canReward
}

func (c *Controller) safeSetDelegateCanDo(do bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.canDelegate = do
}

func (c *Controller) safeGetDelegateCanDo() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.canDelegate
}

func (c *Controller) safeAddDelegateCycle() {
	atomic.AddInt64(&c.delegateCycle, 1)
}

func (c *Controller) safeAddRewardCycle() {
	atomic.AddInt64(&c.rewardCycle, 1)
}

// Start ...
func (c *Controller) Start() {
	go c.WithdrawReward()
	go c.RunDelegate()

	c.Loop()
}

func (c *Controller) Stop() (err error) {
	c.cancel()
	<-time.After(time.Second)
	return
}

func (c *Controller) Loop() {
	t := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-c.ctx.Done():
			klog.Info("[Loop] Received stop signal, exited")
			return
		case <-t.C:
			cycle := c.currentCycle()
			klog.Infof("[Loop] current cycle %d, controller rewardCycle %d, delegateCycle %d", cycle, c.rewardCycle, c.delegateCycle)
			if cycle == c.rewardCycle {
				c.safeSetRewardCanDo(true)
			} else {
				c.safeSetRewardCanDo(false)
			}

			if cycle == c.delegateCycle {
				c.safeSetDelegateCanDo(true)
			} else {
				c.safeSetDelegateCanDo(false)
			}

		}
	}
}
