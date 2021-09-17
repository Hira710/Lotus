package sectorstorage

import (
	"sync"

	sealtasks "github.com/filecoin-project/lotus/extern/sector-storage/sealtasks"
	"github.com/filecoin-project/lotus/extern/sector-storage/storiface"
)

func (a *activeResources) withResources(id WorkerID, wr storiface.WorkerInfo, r Resources, locker sync.Locker, cb func() error) error {
	for !a.canHandleRequest(r, id, "withResources", wr) {
		if a.cond == nil {
			a.cond = sync.NewCond(locker)
		}
		a.cond.Wait()
	}

	a.add(wr.Resources, r)

	err := cb()

	a.free(wr.Resources, r)
	if a.cond != nil {
		a.cond.Broadcast()
	}

	return err
}

// ------------
func (a *activeResources) add(wr storiface.WorkerResources, r Resources) {
	if r.CanGPU {
		a.gpuUsedNum++
		a.gpuUsed = true
	}

	switch r.taskType {
	case sealtasks.TTAddPiece:
		a.apParallelNum++
	case sealtasks.TTPreCommit1:
		a.p1ParallelNum++
	case sealtasks.TTPreCommit2:
		a.p2ParallelNum++
	}

	a.cpuUse += r.Threads(wr.CPUs)
	a.memUsedMin += r.MinMemory
	a.memUsedMax += r.MaxMemory
}

// --------------
func (a *activeResources) free(wr storiface.WorkerResources, r Resources) {
	if r.CanGPU {
		a.gpuUsedNum--
		if a.gpuUsedNum == 0 {
			a.gpuUsed = false
		}
	}

	switch r.taskType {
	case sealtasks.TTAddPiece:
		a.apParallelNum--
	case sealtasks.TTPreCommit1:
		a.p1ParallelNum--
	case sealtasks.TTPreCommit2:
		a.p2ParallelNum--
	}

	a.cpuUse -= r.Threads(wr.CPUs)
	a.memUsedMin -= r.MinMemory
	a.memUsedMax -= r.MaxMemory
}

// canHandleRequest evaluates if the worker has enough available resources to
// handle the request.
func (a *activeResources) canHandleRequest(needRes Resources, wid WorkerID, caller string, info storiface.WorkerInfo) bool {
	if info.IgnoreResources {
		// shortcircuit; if this worker is ignoring resources, it can always handle the request.
		return true
	}

	res := info.Resources
	// TODO: dedupe needRes.BaseMinMemory per task type (don't add if that task is already running)
	minNeedMem := res.MemReserved + a.memUsedMin + needRes.MinMemory + needRes.BaseMinMemory
	if minNeedMem > res.MemPhysical {
		log.Debugf("sched: not scheduling on worker %s for %s; not enough physical memory - need: %dM, have %dM", wid, caller, minNeedMem/mib, res.MemPhysical/mib)
		return false
	}

	maxNeedMem := res.MemReserved + a.memUsedMax + needRes.MaxMemory + needRes.BaseMinMemory

	if maxNeedMem > res.MemSwap+res.MemPhysical {
		log.Debugf("sched: not scheduling on worker %s for %s; not enough virtual memory - need: %dM, have %dM", wid, caller, maxNeedMem/mib, (res.MemSwap+res.MemPhysical)/mib)
		return false
	}

	if a.cpuUse+needRes.Threads(res.CPUs) > res.CPUs {
		log.Debugf("sched: not scheduling on worker %s for %s; not enough threads, need %d, %d in use, target %d", wid, caller, needRes.Threads(res.CPUs), a.cpuUse, res.CPUs)
		return false
	}

	// ------------------------------------------------------------------------
	 if len(res.GPUs) > 0 && needRes.CanGPU { // Meanless
	 	// if a.gpuUsed {
	 	// 	log.Debugf("sched[C2]: not scheduling on worker %s for %s; GPU in use", wid, caller)
	 	// 	return false
	 	// }
		 if len(res.GPUs) >= a.gpuUsedNum {
			log.Debugf("sched[C2]: not scheduling on worker %s for %s; GPU in use", wid, caller)
			return false
		 }
	}

	
	switch needRes.taskType {
	 	case sealtasks.TTAddPiece:
	 		if a.apParallelNum >= LO_AP_PARALLEL_NUM {
	 			// When the worker was filled by P1, there is no need to get AP.
	 			log.Debugf("sched[AP]: not scheduling on worker %s for %s; P1ParallelNum get max", wid, caller)
	 			return false
	 		}

		case sealtasks.TTPreCommit1:
			if a.p1ParallelNum >= LO_P1_PARALLEL_NUM {
				log.Debugf("sched[P1]: not scheduling on worker %s for %s; P1ParallelNum get max", wid, caller)
				return false
			}
	}
	// ------------------------------------------------------------------------

	return true
}

func (a *activeResources) utilization(wr storiface.WorkerResources) float64 {
	var max float64

	cpu := float64(a.cpuUse) / float64(wr.CPUs)
	max = cpu

	memMin := float64(a.memUsedMin+wr.MemReserved) / float64(wr.MemPhysical)
	if memMin > max {
		max = memMin
	}

	memMax := float64(a.memUsedMax+wr.MemReserved) / float64(wr.MemPhysical+wr.MemSwap)
	if memMax > max {
		max = memMax
	}

	return max
}

func (wh *workerHandle) utilization() float64 {
	wh.lk.Lock()
	u := wh.active.utilization(wh.info.Resources)
	u += wh.preparing.utilization(wh.info.Resources)
	wh.lk.Unlock()
	wh.wndLk.Lock()
	for _, window := range wh.activeWindows {
		u += window.allocated.utilization(wh.info.Resources)
	}
	wh.wndLk.Unlock()

	return u
}
