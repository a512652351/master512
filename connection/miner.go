package connection

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type DownstreamClientMutexWrapper struct {
	sync.RWMutex
	clients []*DownstreamClient
}

func (d *DownstreamClientMutexWrapper) GetClients() []*DownstreamClient {
	d.Lock()
	defer d.Unlock()

	return d.clients
}

func (d *DownstreamClientMutexWrapper) Add(c *DownstreamClient) {
	d.Lock()
	defer d.Unlock()

	d.clients = append(d.clients, c)
}

func (d *DownstreamClientMutexWrapper) GetIndex(c *DownstreamClient) int {
	d.Lock()
	defer d.Unlock()

	for i, client := range d.clients {
		if client == c {
			return i
		}
	}

	return -1
}

func (d *DownstreamClientMutexWrapper) Range(f func(i int, c *DownstreamClient) bool) {
	d.Lock()
	defer d.Unlock()

	for i, client := range d.clients {
		if !f(i, client) {
			return
		}
	}
}

func (d *DownstreamClientMutexWrapper) Contains(c *DownstreamClient) bool {
	return d.GetIndex(c) != -1
}

func (d *DownstreamClientMutexWrapper) Remove(c *DownstreamClient) {
	index := d.GetIndex(c)
	if index == -1 {
		return
	}

	d.Lock()
	defer d.Unlock()

	var result []*DownstreamClient
	result = append(result, d.clients[:index]...)
	result = append(result, d.clients[index+1:]...)
	d.clients = result
}

type MinerIdentifier struct {
	Wallet     string
	WorkerName string
}

type WorkerMinerSliceWrapper struct {
	sync.RWMutex
	workerMiner []*WorkerMiner
}

func (wrapper *WorkerMinerSliceWrapper) Range(f func(i int, m *WorkerMiner) bool) {
	wrapper.Lock()
	defer wrapper.Unlock()

	for i, miner := range wrapper.workerMiner {
		if !f(i, miner) {
			return
		}
	}
}

func (wrapper *WorkerMinerSliceWrapper) GetJobIndex(dw *WorkerMiner) int {
	wrapper.Lock()
	defer wrapper.Unlock()

	for i, w := range wrapper.workerMiner {
		if w == dw {
			return i
		}
	}

	return -1
}

func (wrapper *WorkerMinerSliceWrapper) HasMiner(w *WorkerMiner) bool {
	return wrapper.GetJobIndex(w) != -1
}

func (wrapper *WorkerMinerSliceWrapper) Add(w *WorkerMiner) {
	wrapper.Lock()
	defer wrapper.Unlock()

	wrapper.workerMiner = append(wrapper.workerMiner, w)
}

func (wrapper *WorkerMinerSliceWrapper) Remove(m *WorkerMiner) {
	index := wrapper.GetJobIndex(m)
	if index == -1 {
		return
	}

	wrapper.Lock()
	defer wrapper.Unlock()

	if index < len(wrapper.workerMiner) {
		copy(wrapper.workerMiner[index:], wrapper.workerMiner[index+1:])
	}

	wrapper.workerMiner[len(wrapper.workerMiner)-1] = nil
	wrapper.workerMiner = wrapper.workerMiner[:len(wrapper.workerMiner)-1]
}

type WalletMiner struct {
	Wallet string

	TotalShare    int64
	TotalFeeShare int64

	// Clients map[workerName]*WorkerMiner
	Clients *sync.Map
}

func (w *WalletMiner) GetOnlineWorkerList() *[]*WorkerMiner {
	list := make([]*WorkerMiner, 0)
	w.Clients.Range(func(key, value interface{}) bool {
		if value.(*WorkerMiner).IsOnline() {
			list = append(list, value.(*WorkerMiner))
		}
		return true
	})
	return &list
}

func (w *WalletMiner) GetWorkerList() *[]*WorkerMiner {
	list := make([]*WorkerMiner, 0)
	w.Clients.Range(func(key, value interface{}) bool {
		list = append(list, value.(*WorkerMiner))
		return true
	})
	return &list
}

type WorkerMiner struct {
	// ????????????????????????
	ConnectAt time.Time

	// ????????????????????????
	LastShareAt time.Time

	PoolServer *PoolServer
	Identifier *MinerIdentifier

	HashRate   int64
	TotalShare int64

	TimeIntervalShareStats *ShareStatsIntervalMap

	// ????????????????????????????????????????????????
	LastFeeAtShare int64

	// map[*FeeStatesClient]int64
	FeeShareIndividual *sync.Map
	LastFeeTime        time.Time

	// ???????????????????????????????????????
	TimeIntervalFeeShareStats *ShareStatsIntervalMap

	DropUpstream bool

	// ??????????????????
	DownstreamClients *DownstreamClientMutexWrapper
}

func (m *WorkerMiner) GetConn() *[]*DownstreamClient {
	result := make([]*DownstreamClient, 0)
	for _, client := range m.DownstreamClients.GetClients() {
		result = append(result, client)
	}
	return &result
}

func (m *WorkerMiner) IsOnline() bool {
	return len(m.DownstreamClients.GetClients()) > 0
}

func (m *WorkerMiner) AddShare(d int64) {
	atomic.AddInt64(&m.TotalShare, d)
	m.TimeIntervalShareStats.AddShare(d)
	m.LastShareAt = time.Now()

	walletMiner, ok := m.PoolServer.WalletMinerDB.Load(m.Identifier.Wallet)
	if !ok {
		return
	}
	atomic.AddInt64(&walletMiner.(*WalletMiner).TotalShare, d)
}

func (m *WorkerMiner) AddFeeShare(d int64) {
	m.TimeIntervalFeeShareStats.AddShare(d)

	walletMiner, ok := m.PoolServer.WalletMinerDB.Load(m.Identifier.Wallet)
	if !ok {
		return
	}
	atomic.AddInt64(&walletMiner.(*WalletMiner).TotalFeeShare, d)
}

// GetHashrateInMhs Hash/s -> MH/s
func (m *WorkerMiner) GetHashrateInMhs() float64 {
	result, err := strconv.ParseFloat(fmt.Sprintf("%.2f", float64(m.HashRate)/1000000), 64)
	if err != nil {
		return 0
	}
	return result
}

func (m *WorkerMiner) GetID() string {
	return m.Identifier.Wallet + "." + m.Identifier.WorkerName
}

type MinerScore struct {
	ScoreWallet   float64
	Score30Min    float64
	Score15Min    float64
	LastFeeTime   float64
	HashrateLevel float64
	FinalScore    float64
}

// CalcScore ????????????
// ????????????????????????????????? ?????????????????????
func (m *WorkerMiner) CalcScore() MinerScore {
	interval30MinShare := float64(m.TimeIntervalShareStats.GetStats(30 * time.Minute).GetShare())
	interval30MinFeeShare := float64(m.TimeIntervalFeeShareStats.GetStats(30 * time.Minute).GetShare())
	interval15MinShare := float64(m.TimeIntervalShareStats.GetStats(15 * time.Minute).GetShare())
	interval15MinFeeShare := float64(m.TimeIntervalFeeShareStats.GetStats(15 * time.Minute).GetShare())

	// 1 - ?????? / (???????????? * ?????? + ??????)
	// P = ?????? | P < 1
	// S = 24 ????????????????????????
	// f(x) | x = 24 ????????????????????????
	// f\left(x\right)\ =1\ -\ \frac{x}{P\cdot S+\ x}
	score30Min := 1 - (interval30MinFeeShare / ((0.025 * interval30MinShare) + interval30MinFeeShare))

	// ????????????/15 ?????????
	score15Min := 1 - (interval15MinFeeShare / ((0.025 * interval15MinShare) + interval15MinFeeShare))

	// ????????????
	walletMinerObj, ok := m.PoolServer.WalletMinerDB.Load(m.Identifier.Wallet)
	scoreWallet := 0.0
	if ok {
		walletMiner := walletMinerObj.(*WalletMiner)
		scoreWallet = 1 - (float64(walletMiner.TotalFeeShare) / ((0.02 * float64(walletMiner.TotalShare)) + float64(walletMiner.TotalFeeShare)))
	}

	// ???????????????????????? | ????????????
	lastFeeTime := time.Since(m.LastFeeTime).Minutes() / 15
	// ???????????????????????????????????????????????? + 7 min
	if m.LastFeeTime.Unix() == 0 {
		lastFeeTime = time.Since(m.ConnectAt.Add(2*time.Minute)).Minutes() / 15
	}
	if lastFeeTime > 1 {
		lastFeeTime = 1
	}
	if lastFeeTime < 0 {
		lastFeeTime = 0
	}

	// ????????????
	hashrateLevelMax := 800.0
	hashrateLevel := m.GetHashrateInMhs() / hashrateLevelMax
	if hashrateLevel > 1 {
		hashrateLevel = 1
	}

	finalScore := score30Min*0.2 + score15Min*0.1 + hashrateLevel*0.08 + lastFeeTime*0.17 + scoreWallet*0.45
	if math.IsNaN(finalScore) {
		finalScore = 0.0
	}

	// ??????????????????????????? 0
	if m.GetHashrateInMhs() == 0 {
		finalScore = 0.0
	}

	return MinerScore{
		Score30Min:    score30Min,
		Score15Min:    score15Min,
		LastFeeTime:   lastFeeTime,
		ScoreWallet:   scoreWallet,
		HashrateLevel: hashrateLevel,
		FinalScore:    finalScore,
	}
}

type ShareStatsIntervalMap struct {
	sync.RWMutex
	sync.Map
}

func (s *ShareStatsIntervalMap) AddShare(d int64) {
	s.Range(func(_, stats interface{}) bool {
		stats.(*ShareStatsInterval).AddShare(d)
		return true
	})
}

func (s *ShareStatsIntervalMap) GetStats(duration time.Duration) *ShareStatsInterval {
	val, _ := s.LoadOrStore(duration, NewShareStatsInterval(duration))
	return val.(*ShareStatsInterval)
}

func (s *ShareStatsIntervalMap) AddStats(stats *ShareStatsInterval) {
	defer s.Unlock()
	s.Lock()
	s.Store(stats.interval, stats)
}

func (s *ShareStatsIntervalMap) AddStatsSlice(stats *[]*ShareStatsInterval) {
	defer s.Unlock()
	s.Lock()
	for _, interval := range *(stats) {
		s.Store(interval.interval, interval)
	}
}

// ShareStatsInterval ?????????????????????????????????
type ShareStatsInterval struct {
	share           int64
	interval        time.Duration
	intervalStartAt time.Time
}

func NewShareStatsInterval(interval time.Duration) *ShareStatsInterval {
	return &ShareStatsInterval{
		interval:        interval,
		intervalStartAt: time.Now(),
	}
}

func (s *ShareStatsInterval) Update() {
	if time.Since(s.intervalStartAt).Seconds() > s.interval.Seconds() {
		s.share = 0
		s.intervalStartAt = time.Now()
		return
	}
}

func (s *ShareStatsInterval) GetShare() int64 {
	s.Update()
	return s.share
}

func (s *ShareStatsInterval) AddShare(d int64) {
	s.Update()
	atomic.AddInt64(&s.share, d)
}
