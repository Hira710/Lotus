<p align="center">
  <a href="https://docs.filecoin.io/" title="Filecoin Docs">
    <img src="documentation/images/lotus_logo_h.png" alt="Project Lotus Logo" width="244" />
  </a>
</p>

<h1 align="center">Project Lotus - 蓮</h1>

<p align="center">
  <a href="https://circleci.com/gh/filecoin-project/lotus"><img src="https://circleci.com/gh/filecoin-project/lotus.svg?style=svg"></a>
  <a href="https://codecov.io/gh/filecoin-project/lotus"><img src="https://codecov.io/gh/filecoin-project/lotus/branch/master/graph/badge.svg"></a>
  <a href="https://goreportcard.com/report/github.com/filecoin-project/lotus"><img src="https://goreportcard.com/badge/github.com/filecoin-project/lotus" /></a>  
  <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.16-blue.svg" /></a>
  <br>
</p>

変更点:
＊　同一WorkerでAP/P1/P2を処理する
＊　最大８個のP1を同時的に処理する。
＊　一個のP1は4coresを利用する（1core：計算　3cores：データー準備）


env:
FIL_PROOFS_MAXIMIZE_CACHING=1
FIL_PROOFS_USE_GPU_COLUMN_BUILDER=1
FIL_PROOFS_USE_GPU_TREE_BUILDER=1
FIL_PROOFS_USE_MULTICORE_SDR=1
FIL_PROOFS_SDR_PARENTS_CACHE_SIZE=1073741824